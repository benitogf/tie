package router

import (
	"net/http"

	"github.com/benitogf/katamari"
)

// Routes definition
func Routes(server *katamari.Server) {
	// Filters
	server.OpenFilter("boxes/*")
	server.OpenFilter("things/*/*/*") // thing/boxid/userid/id
	server.OpenFilter("mails/*")
	server.OpenFilter("posts/*")
	server.ReadFilter("posts/*", blogFilter)
	server.OpenFilter("stocks/*/*")
	server.OpenFilter("market/*")

	server.Router.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		blogStream(server, w, r)
	}).Methods("GET")
	monitor(server)
}
