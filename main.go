package main

import (
	"flag"
	"log"
	"time"

	"github.com/benitogf/samo"
	"github.com/benitogf/tie/auth"
	"github.com/gorilla/mux"
)

var key = flag.String("key", "a-secret-key", "secret key for tokens")
var path = flag.String("path", "data/auth", "user storage path")

func main() {
	// create a separate data store for auth
	dataStore := &samo.LevelDbStorage{
		Path:    *path,
		Storage: &samo.Storage{Active: false},
	}
	err := dataStore.Start("/")
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
	server.Audit = tokenAuth.Verify
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Start("localhost:8800")
	server.WaitClose()
}
