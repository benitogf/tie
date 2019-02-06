package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/benitogf/samo"
	"github.com/benitogf/tie/auth"
	"github.com/gorilla/mux"
)

func TestRegisterAndAuthorize(t *testing.T) {
	dataStore := &samo.MemoryStorage{
		Memdb:   make(map[string][]byte),
		Storage: &samo.Storage{Active: false}}
	err := dataStore.Start("/")
	if err != nil {
		log.Fatal(err)
	}
	tokenAuth := auth.NewTokenAuth(
		auth.NewJwtStore("a-secret-key", time.Minute*10),
		dataStore,
	)
	server := &samo.Server{}
	server.Silence = true
	server.Audit = tokenAuth.Verify
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Start("localhost:9060")
	defer server.Close(os.Interrupt)

	// unauthorized
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Request creation failed %s", err.Error())
	}
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()

	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusUnauthorized, response.StatusCode)
	}

	// register
	payload := []byte(`{
        "name": "Admin",
        "account":"admin",
        "password": "000",
        "email": "admin@admin.test",
        "phone": "123123123"
    }`)
	req, err = http.NewRequest("POST", "/register", bytes.NewBuffer(payload))
	if err != nil {
		t.Errorf("Request creation failed %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.StatusCode)
	}

	// authorize
	payload = []byte(`{"account":"admin","password":"000"}`)
	req, err = http.NewRequest("POST", "/authorize", bytes.NewBuffer(payload))
	if err != nil {
		t.Errorf("Request creation failed %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.StatusCode)
	}

	dec := json.NewDecoder(response.Body)
	var c auth.Credentials
	if err := dec.Decode(&c); err != nil {
		t.Error("error decoding authorize response")
	}
	if c.Token == "" {
		t.Errorf("Expected a token in the credentials response %s", c)
	}

	token := c.Token
	// log.Println("generated token: ", token)

	// authorized
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Got error on restricted endpoint %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if string(body) != "{\"keys\":[]}" {
		t.Errorf("Expected an empty array. Got %s", string(body))
	}
}
