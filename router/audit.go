package router

import (
	"net/http"
	"strings"

	"github.com/benitogf/katamari/auth"
	"github.com/gorilla/mux"
)

type audits map[string][]string
type customCheck func(path []string, role string, account string) bool
type customAudit map[string]customCheck
type customAudits map[string]customAudit

func auditCheck(list []string, search string) bool {
	result := false
	for _, v := range list {
		if v == search {
			result = true
			break
		}
	}
	return result
}

func admin(path []string, role string, account string) bool {
	return role == "admin"
}

func userThings(path []string, role string, account string) bool {
	return len(path) >= 2 && path[2] == account
}

// no token required
var public = audits{
	"GET":    {},
	"POST":   {"mails"},
	"DELETE": {},
	"PUT":    {},
}

// token required
var private = audits{
	"GET":    {"boxes", "stocks", "market"},
	"POST":   {},
	"DELETE": {},
	"PUT":    {},
}

// custom rules based on role, account or path
var custom = customAudits{
	"GET": {
		"things": userThings,
		"posts":  admin,
		"mails":  admin,
	},
	"POST": {
		"things": userThings,
		"boxes":  admin,
		"posts":  admin,
	},
	"DELETE": {
		"things": userThings,
		"boxes":  admin,
		"posts":  admin,
		"mails":  admin,
	},
	"PUT": {},
}

// Audit requests middleware
func Audit(r *http.Request, auth *auth.TokenAuth) bool {
	key := mux.Vars(r)["key"]
	path := strings.Split(key, "/")
	base := path[0]
	// public endpoints
	if auditCheck(public[r.Method], base) {
		return true
	}

	role, account, err := auth.Audit(r)
	// reject unauthorized
	if err != nil {
		return false
	}

	// clock/keys route
	if r.URL.Path == "/" {
		return true
	}

	// root open authorization
	if role == "root" {
		return true
	}

	// private endpoints
	if auditCheck(private[r.Method], base) {
		return true
	}

	// custom audit endpoints
	if custom[r.Method][base] != nil && custom[r.Method][base](path, role, account) {
		return true
	}

	// default to unauthorized
	return false
}
