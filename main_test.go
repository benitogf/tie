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
	var c auth.Credentials
	dataStore := &samo.MemoryStorage{
		Storage: &samo.Storage{Active: false}}
	err := dataStore.Start()
	if err != nil {
		log.Fatal(err)
	}
	tokenAuth := auth.NewTokenAuth(
		auth.NewJwtStore("a-secret-key", time.Second*1),
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

	dec := json.NewDecoder(response.Body)

	err = dec.Decode(&c)
	if err != nil {
		t.Error("error decoding authorize response")
	}
	if c.Token == "" {
		t.Errorf("Expected a token in the credentials response %s", c)
	}

	regToken := c.Token

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

	dec = json.NewDecoder(response.Body)
	err = dec.Decode(&c)
	if err != nil {
		t.Error("error decoding authorize response")
	}
	if c.Token == "" {
		t.Errorf("Expected a token in the credentials response %s", c)
	}

	token := c.Token
	if token == regToken {
		t.Errorf("Expected register and authorize to provide different tokens")
	}

	// wait expiration of the token
	time.Sleep(time.Second * 2)

	// expired
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Got error on restricted endpoint %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusUnauthorized, response.StatusCode)
	}

	// fake
	fakeToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NDk1MzIxNjM5NDIxMDcxMDAsImlzcyI6ImFkbWluIn0.ZOPToC1AJs1hJRLoyFNZetsxvUNadYNtlIqWrm0FAKE"
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Got error on restricted endpoint %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+fakeToken)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusUnauthorized, response.StatusCode)
	}

	// refresh
	payload = []byte(`{"account":"admin","token":"` + token + `"}`)
	req, err = http.NewRequest("PUT", "/authorize", bytes.NewBuffer(payload))
	if err != nil {
		t.Errorf("Request creation failed %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.StatusCode)
	}

	dec = json.NewDecoder(response.Body)
	err = dec.Decode(&c)
	if err != nil {
		t.Error("error decoding authorize refresh response")
	}
	if c.Token == "" {
		t.Errorf("Expected a token in the refresh credentials response %s", c)
	}

	token = c.Token

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
