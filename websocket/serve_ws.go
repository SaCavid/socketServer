package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"socketserver/models"
	"time"
)

// ServeWs endpoint function which will serve websocket connections
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	log.Println("Attempt to join websocket server")

	// create pool unit for upgrading http to websocket connection
	pool := pool{
		w:  w,
		r:  r,
		Ch: make(chan *websocket.Conn, 1),
	}

	// 60 second timeout for pool workers to do there job
	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
	}()

	hub.pool <- pool
	conn := &websocket.Conn{}

	// wait response from pool about upgrade
	for {
		select {
		case conn = <-pool.Ch:
		case <-ticker.C:
			log.Println("Stopped because of timeout")
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
		if conn.RemoteAddr().String() != "" {
			break
		}
	}

	// after upgrade create client unit for inmemory database registration (HUB)
	client := &Client{
		Id:      fmt.Sprintf("%s", conn.RemoteAddr().String()),
		hub:     hub,
		conn:    conn,
		Send:    make(chan *models.Protocol, 10),
		HubChan: make(chan bool, 1),
	}

	// log.Println("Admin user registered with ID:", conn.RemoteAddr())

	// Send client unit to registration channel in HUB
	client.hub.registerAdmin <- client

	confirm := <-client.HubChan

	// confirm if registration was successfully
	if confirm {
		close(client.HubChan)
	} else {
		close(client.HubChan)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// websocket workers
	go client.writePump()
	client.readPump()
}
