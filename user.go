package main

import (
    "log"
  	"fmt"
    "time"
    "net/http"
  	"encoding/json"
    "github.com/benitogf/pasticho/auth"
)

type User struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	Account  string    `json:"account"`
	Password string    `json:"password"`
	Host     string    `json:"host"`
	Token    string    `json:"token"`
	Expiry   string    `json:"expiry"`
	Start    time.Time `json:"start"`
}

var tokenAuth = auth.NewTokenAuth(nil, nil, jwtstore, nil)
var users []User

/* set secret and default expiration time for tokens */
var jwtstore = auth.NewJwtStore("my-secret-key", time.Minute*10)

func GetUser(account string) User {
	t := jwtstore.NewToken("")
	t.SetClaim("id", account)
  var user User
	user.Name = "Pasticho"
	user.Id = "000"
	user.Account = "pasticho"
	user.Password = "202cb962ac59075b964b07152d234b70"
	user.Token = t.String()
	return user
}

func GetCredentials(w http.ResponseWriter, req *http.Request)  map[string]interface{} {
	dec := json.NewDecoder(req.Body)
	var credentials map[string]interface{}
	if err := dec.Decode(&credentials); err != nil {
		log.Println(err)
		w.WriteHeader(500)
	}
	return credentials
}

func (a *App) authorize(w http.ResponseWriter, req *http.Request) {
  switch req.Method {
    default:
      fmt.Fprintf(w, "Method not suported")
    case "POST":
      w.Header().Add("content-type", "application/json")
      credentials := GetCredentials(w, req)
      if (credentials["account"] == "pasticho" && credentials["password"] == "202cb962ac59075b964b07152d234b70") {
        user := GetUser(credentials["account"].(string))
        enc := json.NewEncoder(w)
        enc.Encode(&user)
      } else {
        respondWithError(w, http.StatusForbidden, "no hay pasticho " + credentials["account"].(string))
      }
    case "PUT":
      w.Header().Add("content-type", "application/json")
      credentials := GetCredentials(w, req)
      if (credentials["token"] != nil) {
        _, err := jwtstore.CheckToken(credentials["token"].(string))
        if err != nil {
          user := GetUser(credentials["account"].(string))
          enc := json.NewEncoder(w)
          enc.Encode(&user)
        } else {
          log.Println("token not expired")
          w.WriteHeader(302)
        }
      } else {
        log.Println("token not found")
        w.WriteHeader(500)
      }
  }
}
