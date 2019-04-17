package main

import (
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
var authPath = flag.String("authPath", "db/auth", "user storage path")
var dataPath = flag.String("dataPath", "db/data", "user storage path")
var port = flag.Int("port", 8800, "service port")

var subscribed = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "active_subscriptions",
	Help: "active subscriptions",
})

func main() {
	prometheus.MustRegister(subscribed)
	// create users storage
	dataStore := &samo.LevelDbStorage{
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

	// Server - Storage
	server.Storage = &samo.LevelDbStorage{
		Path: *dataPath}

	// Server - Audits
	server.Audit = func(r *http.Request) bool {
		// https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
		if r.Header.Get("Upgrade") == "websocket" && r.Header.Get("Sec-WebSocket-Protocol") != "" {
			r.Header.Add("Authorization", "Bearer "+strings.Replace(r.Header.Get("Sec-WebSocket-Protocol"), "bearer, ", "", 1))
		}
		key := mux.Vars(r)["key"]
		authorized := tokenAuth.Verify(r)
		if authorized {
			return true
		}

		if strings.Split(key, "/")[0] == "boxes" && r.Method == "GET" {
			return true
		}

		return false
	}
	server.AuditEvent = func(r *http.Request, event samo.Message) bool {
		return tokenAuth.Verify(r)
	}

	// Server - Monitoring
	server.Subscribe = func(mode string, key string, remoteAddr string) error {
		subscribed.Add(1)
		return nil
	}
	server.Unsubscribe = func(mode string, key string, remoteAddr string) {
		subscribed.Sub(1)
	}

	// Server - Filters
	server.ReceiveFilter("boxes", func(index string, data []byte) ([]byte, error) {
		return data, nil
	})
	server.SendFilter("boxes", func(index string, data []byte) ([]byte, error) {
		return data, nil
	})

	server.ReceiveFilter("boxes/*/*", func(index string, data []byte) ([]byte, error) {
		return data, nil
	})
	server.SendFilter("boxes/*/*", func(index string, data []byte) ([]byte, error) {
		return data, nil
	})

	server.ReceiveFilter("boxes/*", func(index string, data []byte) ([]byte, error) {
		return data, nil
	})
	server.SendFilter("boxes/*", func(index string, data []byte) ([]byte, error) {
		return data, nil
	})

	// Server - Routes
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/profile", tokenAuth.Profile)
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Router.Handle("/metrics", promhttp.Handler())
	server.Start("localhost:" + strconv.Itoa(*port))
	server.WaitClose()
}
