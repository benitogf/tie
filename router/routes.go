package router

import (
	"github.com/gorilla/mux"
	"net/http"
	"github.com/benitogf/katamari"
)

// Routes definition
func Routes(server *katamari.Server) {
	// Filters
	katamari.OpenFilter(server, "boxes/*")
	katamari.OpenFilter(server, "things/*/*/*") // thing/boxid/userid/id
	katamari.OpenFilter(server, "mails/*")
	katamari.OpenFilter(server, "posts/*")
	server.ReadFilter("blog", blogFilter)
	katamari.OpenFilter(server, "stocks/*/*")
	katamari.OpenFilter(server, "market/*")

	// Server - Routes
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		blogStream(server, w, r)
	}).Methods("GET")
	monitor(server)
}