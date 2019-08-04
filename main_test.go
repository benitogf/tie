package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benitogf/samo"
	"github.com/benitogf/tie/auth"
)

func TestRegisterAndAuthorize(t *testing.T) {
	var c auth.Credentials
	dataStore := &samo.MemoryStorage{}
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
	server.Defaults()
	server.Router.HandleFunc("/authorize", tokenAuth.Authorize)
	server.Router.HandleFunc("/profile", tokenAuth.Profile)
	server.Router.HandleFunc("/users", tokenAuth.Users).Methods("GET")
	server.Router.HandleFunc("/user/{account:[a-zA-Z\\d]+}", tokenAuth.User).Methods("GET", "POST", "DELETE")
	server.Router.HandleFunc("/register", tokenAuth.Register).Methods("POST")
	server.Router.HandleFunc("/available", tokenAuth.Available).Queries("account", "{[a-zA-Z\\d]}").Methods("GET")
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
        "name": "root",
        "account":"root",
        "password": "000",
        "email": "root@root.test",
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
	payload = []byte(`{"account":"root","password":"000"}`)
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

	// taken
	req, err = http.NewRequest("GET", "/available?account=root", nil)
	if err != nil {
		t.Errorf("Got error on available endpoint %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusConflict {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusConflict, response.StatusCode)
	}

	//available
	req, err = http.NewRequest("GET", "/available?account=notadmin", nil)
	if err != nil {
		t.Errorf("Got error on available endpoint %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.StatusCode)
	}

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

	// refresh user doesn't match token
	payload = []byte(`{"account":"notadmin","token":"` + token + `"}`)
	req, err = http.NewRequest("PUT", "/authorize", bytes.NewBuffer(payload))
	if err != nil {
		t.Errorf("Request creation failed %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusBadRequest, response.StatusCode)
	}

	// refresh
	payload = []byte(`{"account":"root","token":"` + token + `"}`)
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
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if string(body) != "{\"keys\":[]}" {
		t.Errorf("Expected an empty array. Got %s", string(body))
	}

	// profile
	req, err = http.NewRequest("GET", "/profile", nil)
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

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if strings.TrimRight(string(body), "\n") != `{"name":"root","email":"root@root.test","phone":"123123123","account":"root","role":"root"}` {
		t.Errorf("Expected the user profile. Got %s", string(body))
	}

	// users
	req, err = http.NewRequest("GET", "/users", nil)
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
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if strings.TrimRight(string(body), "\n") != `[{"name":"root","email":"root@root.test","phone":"123123123","account":"root","role":"root"}]` {
		t.Errorf("Expected the user profile. Got %s", string(body))
	}

	// get user
	req, err = http.NewRequest("GET", "/user/root", nil)
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
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if strings.TrimRight(string(body), "\n") != `{"name":"root","email":"root@root.test","phone":"123123123","account":"root","role":"root"}` {
		t.Errorf("Expected the user profile. Got %s", string(body))
	}

	// update user
	payload = []byte(`{"phone":"321321321"}`)
	req, err = http.NewRequest("POST", "/user/root", bytes.NewBuffer(payload))
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
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if strings.TrimRight(string(body), "\n") != `{"name":"root","email":"root@root.test","phone":"321321321","account":"root","role":"root"}` {
		t.Errorf("Expected the user profile. Got %s", string(body))
	}

	// get updated user
	req, err = http.NewRequest("GET", "/user/root", nil)
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
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if strings.TrimRight(string(body), "\n") != `{"name":"root","email":"root@root.test","phone":"321321321","account":"root","role":"root"}` {
		t.Errorf("Expected the user profile. Got %s", string(body))
	}

	// delete user
	req, err = http.NewRequest("DELETE", "/user/root", nil)
	if err != nil {
		t.Errorf("Got error on restricted endpoint %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusNoContent {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusNoContent, response.StatusCode)
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Got error reading response.  %s", err.Error())
	}
	if strings.TrimRight(string(body), "\n") != `deleted root` {
		t.Errorf("Expected deleted message. Got %s", string(body))
	}

	// available deleted user
	req, err = http.NewRequest("GET", "/available?account=root", nil)
	if err != nil {
		t.Errorf("Got error on available endpoint %s", err.Error())
	}
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.StatusCode)
	}
}
