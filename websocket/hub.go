package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Resolve cross-domain problems
		// FIXME
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	workers     = os.Getenv("")
	broadcaster = os.Getenv("")
)

type Protocol struct {
	To      string
	Command string

	Error     bool
	ErrCode   uint32
	Msg       string
	AdminChan chan *Protocol
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

	if p.To == "" {
		return fmt.Errorf("machine id cant be null")
	}

	if len(p.Command) == 0 || len(p.Command) > 30 {
		return fmt.Errorf("wrong command format")
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

	// Registered admin users.
	users map[string]*Client

	// Registered machines.
	machines map[string]*Client

	// Inbound messages from the machines.
	broadcast chan *Protocol

	// Register requests from the machines.
	registerAdmin chan *Client

	// Unregister requests from machines.
	unregisterAdmin chan *Client

	// Register requests from the machines.
	Register chan *Client

	// Unregister requests from machines.
	Unregister chan *Client
}

func NewHub() *Hub {

	bc, err := strconv.ParseInt(broadcaster, 10, 64)
	if err != nil {
		bc = 256
	}

	wr, err := strconv.ParseInt(workers, 10, 64)
	if err != nil {
		wr = 128
	}

	return &Hub{
		broadcast:       make(chan *Protocol, bc),
		users:           make(map[string]*Client),
		registerAdmin:   make(chan *Client),
		unregisterAdmin: make(chan *Client),
		machines:        make(map[string]*Client),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		pool:            make(chan pool, wr),
	}
}

func (h *Hub) Run() {

	bc, err := strconv.ParseInt(broadcaster, 10, 64)
	if err != nil {
		bc = 256
	}

	for i := bc; i >= 0; i-- {
		go h.Broadcast()
	}

	go h.Informer()

	for {
		select {
		case client := <-h.Register:
			//			log.Println("Register new machine client in HUB")
			h.rw.Lock()
			_, ok := h.machines[client.Id]
			if ok {
				//	log.Println(fmt.Sprintf("Machine with id: %s already exists", client.Id))
				client.HubChan <- false
			} else {
				h.machines[client.Id] = client
				client.HubChan <- true
			}
			h.rw.Unlock()
		case client := <-h.Unregister:
			h.rw.RLock()
			_, ok := h.machines[client.Id]
			h.rw.RUnlock()
			if ok {
				//				log.Println("Unregister machine client")
				h.rw.Lock()
				delete(h.machines, client.Id)
				h.rw.Unlock()
			}
		case client := <-h.registerAdmin:
			log.Println("Register new Admin client")
			h.rw.Lock()
			h.users[client.Id] = client
			h.rw.Unlock()
		case client := <-h.unregisterAdmin:
			log.Println("Unregister Admin client")
			h.rw.RLock()
			_, ok := h.users[client.Id]
			h.rw.RUnlock()
			if ok {
				h.rw.Lock()
				delete(h.users, client.Id)
				h.rw.Unlock()
				close(client.Send)
			}
		}
	}
}

func (h *Hub) Broadcast() {
	// broadcasts message between connected clients (users --> machines)
	for {
		select {
		case initializer := <-h.broadcast:
			h.rw.RLock()

			machine, ok := h.machines[initializer.To]
			if ok {
				machine.Send <- initializer
				initializer.Msg = fmt.Sprintf("Command to %s sent successfully", initializer.To)
				initializer.AdminChan <- initializer
			} else {
				log.Println(fmt.Sprintf("Error: Client with ID: %s not found", initializer.To))
				initializer.Error = true
				initializer.Msg = fmt.Sprintf("Client with ID: %s not found", initializer.To)
				initializer.AdminChan <- initializer
			}
			h.rw.RUnlock()
		}
	}
}

func (h *Hub) Informer() {
	ticker := time.NewTicker(informerPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			h.rw.RLock()
			machines := len(h.machines)
			log.Println(runtime.NumGoroutine())
			for _, v := range h.users {

				msg := fmt.Sprintf("There is %d machine online", machines)

				if machines > 1 {
					msg = fmt.Sprintf("There are %d machines online", machines)
				}

				err := v.conn.WriteJSON(&struct {
					ErrCode int
					Msg     string
				}{
					ErrCode: 1,
					Msg:     msg,
				})

				if err != nil {
					log.Println(err)
					h.unregisterAdmin <- v
				}
			}

			h.rw.RUnlock()
		}
	}
}

func (h *Hub) Pool() {

	wr, err := strconv.ParseInt(workers, 10, 64)
	if err != nil {
		wr = 128
	}

	wg := &sync.WaitGroup{}
	wg.Add(int(wr))
	// create 256 workers for websocket/tcp registration
	var o int
	for i := 1; i <= int(wr); i++ {
		go h.worker()
		o++
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
