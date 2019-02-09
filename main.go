package main

import (
	"flag"
	"log"
	"strconv"
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
		Path:    *authPath,
		Storage: &samo.Storage{Active: false},
	}
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
	server.Silence = false
	server.Storage = &samo.LevelDbStorage{
		Path:    *dataPath,
		Storage: &samo.Storage{Active: false},
	}
	server.Audit = tokenAuth.Verify
	server.Subscribe = func(mode string, key string, remoteAddr string) error {
		subscribed.Add(1)
		return nil
	}
	server.Unsubscribe = func(mode string, key string, remoteAddr string) {
		subscribed.Sub(1)
	}
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Router.Handle("/metrics", promhttp.Handler())
	server.Start("localhost:" + strconv.Itoa(*port))
	server.WaitClose()
}
