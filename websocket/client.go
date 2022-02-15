package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"time"
)

const (

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Send pings to peer with this period. Must be less than pongWait.
	informerPeriod = 5 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Id string // Client unique id: remote address for unique connection

	hub *Hub

	// channel to confirm registration
	HubChan chan bool

	// The websocket connection.
	conn *websocket.Conn

	TConn net.Conn
	// Buffered channel of outbound messages.
	Send chan *Protocol
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregisterAdmin <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {

		msg := &Protocol{}

		err := c.conn.ReadJSON(msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			log.Println(err)
			break
		} else {

			err = msg.Validate()
			if err != nil {
				log.Println(err)
				c.Send <- &Protocol{
					Error:   true,
					ErrCode: 0,
					Msg:     err.Error(),
				}
			} else {
				msg.AdminChan = c.Send
				c.hub.broadcast <- msg
			}
		}

	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// The hub closed the channel.
				log.Println("Closed channel:")
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println(err)
				return
			}

			text, err := json.Marshal(&struct {
				To      string
				Command string

				Error   bool
				ErrCode uint32
				Msg     string
			}{
				To:      message.To,
				Command: message.Command,
				Error:   message.Error,
				ErrCode: message.ErrCode,
				Msg:     message.Msg,
			})

			if err != nil {
				log.Println(err)
				return
			}

			_, _ = w.Write(text)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	log.Println("Attempt to join websocket server")
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

	client := &Client{
		Id:   fmt.Sprintf("%s", conn.RemoteAddr().String()),
		hub:  hub,
		conn: conn,
		Send: make(chan *Protocol, 10),
	}

	log.Println("Admin user registered with ID:", conn.RemoteAddr().String())
	log.Println("Admin user registered with ID:", conn.RemoteAddr().Network())

	client.hub.registerAdmin <- client

	go client.writePump()
	go client.readPump()
}
