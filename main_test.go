package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/katamari/messages"
	"github.com/gorilla/websocket"

	"github.com/benitogf/auth"
	"github.com/benitogf/katamari"
	"github.com/benitogf/tie/router"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestBlog(t *testing.T) {
	// var c auth.Credentials
	var wg sync.WaitGroup
	var wsEvent messages.Message
	var mutex sync.Mutex
	authStore := &katamari.MemoryStorage{}
	err := authStore.Start(katamari.StorageOpt{})
	require.NoError(t, err)
	go katamari.WatchStorageNoop(authStore)
	auth := auth.New(
		auth.NewJwtStore("a-secret-key", time.Second*1),
		authStore,
	)
	server := &katamari.Server{}
	server.Silence = true
	server.Audit = func(r *http.Request) bool {
		return router.Audit(r, auth)
	}
	server.Router = mux.NewRouter()
	router.Routes(server)
	auth.Router(server)
	server.Start("localhost:9978")
	defer server.Close(os.Interrupt)

	req, err := http.NewRequest("GET", "/posts/*", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)

	req, err = http.NewRequest("DELETE", "/posts/*", nil)
	require.NoError(t, err)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response = w.Result()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)

	wsURL := url.URL{Scheme: "ws", Host: "localhost:9978", Path: "/blog"}
	wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil {
				break
			}
			mutex.Lock()
			wsEvent, err = messages.DecodeTest(message)
			require.NoError(t, err)
			mutex.Unlock()
			wg.Done()
		}
	}()
	wg.Wait()
	require.Equal(t, "[]", wsEvent.Data)
}
