package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	workers     = 256
	broadcaster = 128
)

type Protocol struct {
	// Type indexes
	// 0 error
	// 11 text 12 images 13 file (conversation)
	// 41 text 42 images 43 file (reply)
	// 50 ack received by server 51 ack received by users 52 ack read by users
	// 53 received by user 54 read by user 55 delete for user 56 delete for everyone if owner asked
	// 61 text 62 images 63 file (from contacts list - team members)
	// 7 email type (only from server - support) - no need to answer
	// 8 forward conversation to other member or support - data can be nil

	Type           uint8
	ConversationId string // similar to topic - can be forwarded to other user
	MessageId      string

	Data Message
}

type Message struct {
	From  string
	To    string
	Reply string

	Text   string
	Images []string
	File   string
}

func (p *Protocol) Validate() error {

	if p.MessageId == "" {
		return fmt.Errorf("message id cant be nil")
	}

	if p.Type > 11 || p.Type < 1 {
		return fmt.Errorf("wrong type index")
	}

	if p.Data.To != "" {
		if p.ConversationId == "" {
			return fmt.Errorf("conversation id cant be nil if message")
		}
	}

	switch p.Type {
	case 1, 41, 8:
		if p.Data.Text == "" {
			return fmt.Errorf("text field is empty")
		}
	case 2, 42:
		if len(p.Data.Images) == 0 {
			return fmt.Errorf("images not included")
		}
	case 3, 43:
		if p.Data.File == "" {
			return fmt.Errorf("file not included")
		}
	case 7:
		if p.Data.To == "" {
			return fmt.Errorf("to field must be filled")
		}

	}

	return nil
}

type pool struct {
	w  http.ResponseWriter
	r  *http.Request
	Ch chan *websocket.Conn
}

type Hub struct {
	pool chan pool

	// write locker
	rw sync.RWMutex

	// Registered clients.
	clients map[string]*Client

	// Inbound messages from the clients.
	broadcast chan *Protocol

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Protocol, broadcaster),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		pool:       make(chan pool, workers),
	}
}

func (h *Hub) Run() {

	for i := broadcaster; i >= 0; i-- {
		go h.Broadcast()
	}

	for {
		select {
		case client := <-h.register:
			h.rw.Lock()
			h.clients[client.Id] = client
			h.rw.Unlock()
		case client := <-h.unregister:

			h.rw.RLock()
			_, ok := h.clients[client.Id]
			h.rw.RUnlock()
			if ok {
				h.rw.Lock()
				delete(h.clients, client.Id)
				h.rw.Unlock()
				close(client.send)
			}
		}
	}
}

func (h *Hub) Broadcast() {

	for {
		select {
		case message := <-h.broadcast:
			h.rw.RLock()
			client, ok := h.clients[message.ConversationId]
			h.rw.RUnlock()
			if ok {
				client.send <- message
			}
		}
	}
}

func (h *Hub) Pool() {
	wg := &sync.WaitGroup{}

	wg.Add(workers)
	// create 256 workers for websocket registration
	for i := workers; i <= 0; i++ {
		go h.worker()
	}

	wg.Wait()
}

func (h *Hub) worker() {
	for {
		select {
		case pool := <-h.pool:

			conn, err := upgrader.Upgrade(pool.w, pool.r, nil)
			if err != nil {
				log.Println(err)
				return
			}

			pool.Ch <- conn

		}
	}
}
