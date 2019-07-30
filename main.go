package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/benitogf/samo"
	"github.com/benitogf/tie/auth"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var key = flag.String("key", "a-secret-key", "secret key for tokens")
var authPath = flag.String("authPath", "db/auth", "auth storage path")
var dataPath = flag.String("dataPath", "db/data", "data storage path")
var port = flag.Int("port", 8800, "service port")

var subscribed = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "active_subscriptions",
	Help: "active subscriptions",
})

func openFilter(index string, data []byte) ([]byte, error) {
	return data, nil
}

func closedFilter(index string, data []byte) ([]byte, error) {
	return data, errors.New("out of bounds route")
}

func addListDetailFilter(server *samo.Server, name string) {
	server.ReceiveFilter(name, openFilter)
	server.SendFilter(name, openFilter)
	server.ReceiveFilter(name+"/*", openFilter)
	server.SendFilter(name+"/*", openFilter)
	server.ReceiveFilter(name+"/*/*", closedFilter)
	server.SendFilter(name+"/*/*", closedFilter)
}

func addRelatedListDetailFilter(server *samo.Server, name string) {
	server.ReceiveFilter(name, closedFilter)
	server.SendFilter(name, closedFilter)
	server.ReceiveFilter(name+"/*", openFilter)
	server.SendFilter(name+"/*", openFilter)
	server.ReceiveFilter(name+"/*/*", openFilter)
	server.SendFilter(name+"/*/*", openFilter)
	server.ReceiveFilter(name+"/*/*/*", openFilter)
	server.SendFilter(name+"/*/*/*", openFilter)
	server.ReceiveFilter(name+"/*/*/*/*", closedFilter)
	server.SendFilter(name+"/*/*/*/*", closedFilter)
}

func main() {
	flag.Parse()

	// prometheus
	prometheus.MustRegister(subscribed)

	// create users storage
	dataStore := &samo.LevelStorage{
		Path: *authPath}
	err := dataStore.Start()
	if err != nil {
		log.Fatal(err)
	}

	// create a tokenAuth
	tokenAuth := auth.NewTokenAuth(
		auth.NewJwtStore(*key, time.Minute*10),
		dataStore,
	)

	// Server
	server := &samo.Server{}
	server.Silence = false // logs silence
	server.Static = true   // only allow filtered paths

	// Storage
	server.Storage = &samo.LevelStorage{
		Path: *dataPath}

	// Audits
	server.Audit = func(r *http.Request) bool {
		key := mux.Vars(r)["key"]
		path := strings.Split(key, "/")
		// public endpoints
		// read only
		if path[0] == "boxes" && r.Method == "GET" {
			return true
		}

		if path[0] == "things" && r.Method == "GET" {
			return true
		}

		if path[0] == "posts" && r.Method == "GET" {
			return true
		}

		// write only
		if path[0] == "mails" && r.Method == "POST" {
			return true
		}

		// get the header from a websocket connection
		// https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
		if r.Header.Get("Upgrade") == "websocket" && r.Header.Get("Sec-WebSocket-Protocol") != "" {
			r.Header.Add("Authorization", "Bearer "+strings.Replace(r.Header.Get("Sec-WebSocket-Protocol"), "bearer, ", "", 1))
		}

		token, err := tokenAuth.Authenticate(r)
		authorized := (err == nil)
		role := "user"
		account := ""
		if authorized {
			role = token.Claims("role").(string)
			account = token.Claims("iss").(string)
		}

		// root and admin authorization
		if authorized && (role == "admin" || role == "root") {
			return true
		}

		// user related things
		if authorized && path[0] == "things" && len(path) >= 2 && path[2] == account {
			return true
		}

		if authorized && r.URL.Path == "/time" {
			return true
		}

		return false
	}

	// Monitoring
	server.Subscribe = func(mode string, key string, remoteAddr string) error {
		subscribed.Add(1)
		return nil
	}
	server.Unsubscribe = func(mode string, key string, remoteAddr string) {
		subscribed.Sub(1)
	}

	// Filters
	addListDetailFilter(server, "boxes")
	addRelatedListDetailFilter(server, "things")
	addListDetailFilter(server, "mails")
	addListDetailFilter(server, "posts")

	// Server - Routes
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/profile", tokenAuth.Profile)
	server.Router.HandleFunc("/users", tokenAuth.Users).Methods("GET")
	server.Router.HandleFunc("/user/{account:[a-zA-Z\\d]+}", tokenAuth.User).Methods("GET", "POST", "DELETE")
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Router.HandleFunc("/available", tokenAuth.Available).Queries("account", "{[a-zA-Z\\d]}").Methods("GET")
	server.Router.Handle("/metrics", promhttp.Handler())
	server.Start("localhost:" + strconv.Itoa(*port))
	server.WaitClose()
}
