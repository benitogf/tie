package main

import (
    "log"
  	"fmt"
  	"net/http"
    "database/sql"
    "github.com/gorilla/mux"
    "github.com/benitogf/pasticho/auth"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    _ "github.com/lib/pq"
)

type App struct {
  Router   *mux.Router
  DB       *sql.DB
}

func (app *App) Initialize(user, password, dbname string) {
    connectionString := fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=disable", user, password, dbname)
    var err error
    app.DB, err = sql.Open("postgres", connectionString)
    if err != nil {
        log.Fatal(err)
    }

    app.Router = mux.NewRouter()
    app.initializeRoutes()
}

func (app *App) Run(addr *string) {
    stdout.Printf("Using %s as address to listen.\n", *addr)
    log.Fatal(http.ListenAndServe(*addr, app.Router))
}

func (app *App) initializeRoutes() {
  app.Router.HandleFunc("/file", tokenAuth.HandleFunc(app.createFile)).Methods("POST")
  app.Router.HandleFunc("/file/{id:[0-9]+}", tokenAuth.HandleFunc(app.getFile)).Methods("GET")
  app.Router.HandleFunc("/file/{id:[0-9]+}", tokenAuth.HandleFunc(app.deleteFile)).Methods("DELETE")
  app.Router.HandleFunc("/products", tokenAuth.HandleFunc(app.getProducts)).Methods("GET")
  app.Router.HandleFunc("/product", tokenAuth.HandleFunc(app.createProduct)).Methods("POST")
  app.Router.HandleFunc("/product/{id:[0-9]+}", tokenAuth.HandleFunc(app.getProduct)).Methods("GET")
  app.Router.HandleFunc("/product/{id:[0-9]+}", tokenAuth.HandleFunc(app.updateProduct)).Methods("PUT")
  app.Router.HandleFunc("/product/{id:[0-9]+}", tokenAuth.HandleFunc(app.deleteProduct)).Methods("DELETE")
  app.Router.HandleFunc("/authorize", app.authorize)
  app.Router.HandleFunc("/restricted", tokenAuth.HandleFunc(app.pasticho))
  app.Router.HandleFunc("/ws", app.wss)
  app.Router.Handle("/metrics", promhttp.Handler())
}

func (a *App) pasticho(w http.ResponseWriter, req *http.Request) {
  token := auth.Get(req)
  fmt.Fprintf(w, "hay %s", token.Claims("id").(string))
}
