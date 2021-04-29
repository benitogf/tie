package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/benitogf/auth"
	"github.com/benitogf/katamari"
	"github.com/benitogf/level"
	"github.com/benitogf/tie/router"
	"github.com/gorilla/mux"
)

var key = flag.String("key", "a-secret-key", "secret key for tokens")
var authPath = flag.String("authPath", "db/auth", "auth storage path")
var dataPath = flag.String("dataPath", "db/data", "data storage path")
var port = flag.Int("port", 8800, "service port")

func main() {
	flag.Parse()

	// auth
	authStore := &level.Storage{Path: *authPath}
	err := authStore.Start(katamari.StorageOpt{})
	if err != nil {
		log.Fatal(err)
	}
	go katamari.WatchStorageNoop(authStore)
	auth := auth.New(
		auth.NewJwtStore(*key, time.Minute*10),
		authStore,
	)

	// Server
	server := &katamari.Server{}
	server.Silence = false
	server.Static = true
	server.Storage = &level.Storage{Path: *dataPath}
	server.Audit = func(r *http.Request) bool {
		return router.Audit(r, auth)
	}
	server.OnClose = func() {
		authStore.Close()
	}
	server.Router = mux.NewRouter()
	router.Routes(server)
	auth.Router(server)
	server.Start("localhost:" + strconv.Itoa(*port))
	server.WaitClose()
}
