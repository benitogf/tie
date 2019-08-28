package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/benitogf/katamari/storages/level"
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

func addOpenFilter(server *katamari.Server, name string) {
	server.WriteFilter(name, openFilter)
	server.ReadFilter(name, openFilter)
}

func auditRequest(r *http.Request, tokenAuth *auth.TokenAuth) bool {
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

	if path[0] == "stocks" && r.Method == "GET" {
		return true
	}

	if path[0] == "market" && r.Method == "GET" {
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

	if authorized && r.URL.Path == "/" {
		return true
	}

	return false
}

func blogFilter(index string, data []byte) ([]byte, error) {
	type post struct {
		Active bool `json:"active"`
	}
	unfiltered, err := objects.DecodeList(data)
	if err != nil {
		return []byte(""), err
	}
	filtered := []objects.Object{}
	for _, obj := range unfiltered {
		var postData post
		err = json.Unmarshal([]byte(obj.Data), &postData)
		if err == nil && postData.Active {
			filtered = append(filtered, obj)
		}
	}
	rawFiltered, err := objects.Encode(filtered)
	if err != nil {
		return []byte(""), err
	}
	return rawFiltered, nil
}

func blogStream(server *katamari.Server, w http.ResponseWriter, r *http.Request) {
	client, err := server.Stream.New("posts/*", "blog", w, r)
	if err != nil {
		return
	}

	entry, err := server.Fetch("posts/*", "blog")
	if err != nil {
		return
	}

	go server.Stream.Write(client, messages.Encode(entry.Data), true, entry.Version)
	server.Stream.Read("posts/*", "blog", client)
}

func watchStorage(dataStore *level.Storage) {
	for {
		_ = <-dataStore.Watch()
		if !dataStore.Active() {
			break
		}
	}
}

func main() {
	flag.Parse()

	// prometheus
	prometheus.MustRegister(subscribed)

	// create users storage
	authStore := &level.Storage{Path: *authPath}
	err := authStore.Start()
	if err != nil {
		log.Fatal(err)
	}

	// create a tokenAuth
	tokenAuth := auth.NewTokenAuth(
		auth.NewJwtStore(*key, time.Minute*10),
		authStore,
	)

	// Server
	server := &katamari.Server{}
	server.Silence = false // logs silence
	server.Static = true   // only allow filtered paths
	go watchStorage(authStore)

	// Storage
	server.Storage = &level.Storage{Path: *dataPath}

	// Audit
	server.Audit = func(r *http.Request) bool {
		return auditRequest(r, tokenAuth)
	}

	// Monitoring
	server.OnSubscribe = func(key string) error {
		subscribed.Add(1)
		return nil
	}
	server.OnUnsubscribe = func(key string) {
		subscribed.Sub(1)
	}

	// Filters
	addOpenFilter(server, "boxes/*")
	addOpenFilter(server, "things/*/*/*") // thing/boxid/userid/id
	addOpenFilter(server, "mails/*")
	addOpenFilter(server, "posts/*")
	server.ReadFilter("blog", blogFilter)
	addOpenFilter(server, "stocks/*/*")
	addOpenFilter(server, "market/*")

	// Server - Routes
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/profile", tokenAuth.Profile)
	server.Router.HandleFunc("/users", tokenAuth.Users).Methods("GET")
	server.Router.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		blogStream(server, w, r)
	}).Methods("GET")
	server.Router.HandleFunc("/user/{account:[a-zA-Z\\d]+}", tokenAuth.User).Methods("GET", "POST", "DELETE")
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Router.HandleFunc("/available", tokenAuth.Available).Queries("account", "{[a-zA-Z\\d]}").Methods("GET")
	server.Router.Handle("/metrics", promhttp.Handler())
	server.Start("localhost:" + strconv.Itoa(*port))
	server.WaitClose()
}
