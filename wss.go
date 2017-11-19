package main

import (
    "log"
    "time"
    "net/http"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool {
    return true
}, }

func (a *App) wss(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
      log.Print(err)
        return
    }
    defer c.Close()
    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            log.Println("read:", err)
            break
        }
        log.Printf("recv: %s", message)
        if (string(message) == "parameter:time") {
            loc, _ := time.LoadLocation("Asia/Macau")
            now := time.Now()
            t := now.In(loc)
            err = c.WriteMessage(mt, []byte(t.String()))
            if err != nil {
                log.Println("write:", err)
                break
            }
        }
    }
}
