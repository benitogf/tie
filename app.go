package main

import (
    "log"
  	"fmt"
    "math"
    "math/rand"
  	"net/http"
    "database/sql"
    "github.com/gorilla/mux"
    "github.com/benitogf/pasticho/auth"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    _ "github.com/lib/pq"
)

type App struct {
  Router   *mux.Router
  DB       *sql.DB
}

// https://godoc.org/github.com/prometheus/client_golang/prometheus#Summary
// A Summary captures individual observations from an event or sample stream and summarizes
// them in a manner similar to traditional summary statistics:
//     1. sum of observations
//     2. observation count
//     3. rank estimations.
var temps = prometheus.NewSummary(prometheus.SummaryOpts{
    Name: "pasticho_temperature_celsius",
    Help: "The temperature of the pasticho while cooking it.",
    Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
})

func (app *App) Initialize(user, password, dbname string) {
    connectionString := fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=disable", user, password, dbname)
    var err error
    app.DB, err = sql.Open("postgres", connectionString)
    if err != nil {
        log.Fatal(err)
    }

    prometheus.MustRegister(temps)
    app.Router = mux.NewRouter()
    app.initializeRoutes()
}

func (app *App) Run(addr *string) {
    stdout.Printf("Using %s as address to listen.\n", *addr)
    log.Fatal(http.ListenAndServe(*addr, app.Router))
}

func (app *App) initializeRoutes() {
  app.Router.HandleFunc("/get/{key}", app.get).Methods("GET")
  // app.Router.HandleFunc("/get/batch", app.getBatch).Methods("GET")
  app.Router.HandleFunc("/exists/{key}", tokenAuth.HandleFunc(app.exists)).Methods("GET")
  app.Router.HandleFunc("/free/{key}", tokenAuth.HandleFunc(app.free)).Methods("GET")
  app.Router.HandleFunc("/set", tokenAuth.HandleFunc(app.set)).Methods("POST")
  // app.Router.HandleFunc("/set/batch", tokenAuth.HandleFunc(app.setBatch)).Methods("POST")
  app.Router.HandleFunc("/del/{key}", tokenAuth.HandleFunc(app.del)).Methods("DELETE")
  // app.Router.HandleFunc("/del/batch", tokenAuth.HandleFunc(app.delBatch)).Methods("DELETE")

  app.Router.HandleFunc("/file", tokenAuth.HandleFunc(app.createFile)).Methods("POST")
  app.Router.HandleFunc("/file/{id:[0-9]+}", tokenAuth.HandleFunc(app.getFile)).Methods("GET")
  app.Router.HandleFunc("/file/{id:[0-9]+}", tokenAuth.HandleFunc(app.deleteFile)).Methods("DELETE")
  app.Router.HandleFunc("/products", tokenAuth.HandleFunc(app.getProducts)).Methods("GET")
  app.Router.HandleFunc("/product", tokenAuth.HandleFunc(app.createProduct)).Methods("POST")
  app.Router.HandleFunc("/product/{id:[0-9]+}", tokenAuth.HandleFunc(app.getProduct)).Methods("GET")
  app.Router.HandleFunc("/product/{id:[0-9]+}", tokenAuth.HandleFunc(app.updateProduct)).Methods("PUT")
  app.Router.HandleFunc("/product/{id:[0-9]+}", tokenAuth.HandleFunc(app.deleteProduct)).Methods("DELETE")
  app.Router.HandleFunc("/authorize", app.authorize)
  app.Router.HandleFunc("/restricted", tokenAuth.HandleFunc(app.protected))
  app.Router.HandleFunc("/temperature", app.temperature)
  app.Router.HandleFunc("/ws", app.wss)
  app.Router.Handle("/metrics", promhttp.Handler())
}

func (a *App) protected(w http.ResponseWriter, req *http.Request) {
  token := auth.Get(req)
  fmt.Fprintf(w, "token: %s", token.Claims("id").(string))
}

func (a *App) temperature(w http.ResponseWriter, req *http.Request) {
  // Simulate one observation
  temp := 30 + math.Floor(9194*math.Sin(rand.Float64()*0.1))/10
  // observe it
  temps.Observe(temp)
  response := fmt.Sprintf("The pasticho is cooking at %.2f°C.\n", temp)
  stdout.Printf(response)
  fmt.Fprintf(w, response)
}
