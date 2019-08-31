package router

import (
	"github.com/benitogf/katamari/auth"
	"strings"
	"github.com/gorilla/mux"
	"net/http"
)

// Audit requests middleware
func Audit(r *http.Request, auth *auth.TokenAuth) bool {
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

	token, err := auth.Authenticate(r)
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