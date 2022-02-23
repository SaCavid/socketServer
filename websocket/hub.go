package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"runtime"
	"socketserver/models"
	"strconv"
	"sync"
	"time"
)

var (
	upgrader = websocket.Upgrader{
		// default buffers
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Resolve cross-domain problems
		// FIXME
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type pool struct {
	w  http.ResponseWriter
	r  *http.Request
	Ch chan *websocket.Conn
}

// Hub main model for registration control
type Hub struct {
	pool chan pool

	// write locker
	rw sync.RWMutex

	// Registered admin Bots.
	Bots map[string]*Client

	// Registered machines.
	machines map[string]*Client

	// Inbound messages from the machines.
	BroadcastChannel chan *models.Protocol

	// Register requests from the websocket.
	RegisterWs chan *Client

	// Unregister requests from websocket.
	UnRegisterWs chan *Client

	// Register requests from the machines.
	RegisterBot chan *Client

	// Unregister requests from machines.
	UnRegisterBot chan *Client

	// Register requests from the machines.
	Register chan *Client

	// Unregister requests from machines.
	Unregister chan *Client
}

// NewHub creates new object from type
func NewHub() *Hub {

	workers := os.Getenv("TCP_HUB_WORKERS")
	broadcaster := os.Getenv("BROADCAST_WORKERS")
	bc, err := strconv.ParseInt(broadcaster, 10, 64)
	if err != nil {
		bc = 256
	}

	wr, err := strconv.ParseInt(workers, 10, 64)
	if err != nil {
		log.Println(err)
		wr = 128
	}

	return &Hub{
		BroadcastChannel: make(chan *models.Protocol, bc),
		Bots:             make(map[string]*Client),
		RegisterWs:       make(chan *Client),
		UnRegisterWs:     make(chan *Client),
		RegisterBot:      make(chan *Client),
		UnRegisterBot:    make(chan *Client),
		machines:         make(map[string]*Client),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		pool:             make(chan pool, wr),
	}
}

// Run Start HUB
func (h *Hub) Run() {

	broadcaster := os.Getenv("BROADCAST_WORKERS")
	bc, err := strconv.ParseInt(broadcaster, 10, 64)
	if err != nil {
		bc = 256
	}

	for i := bc; i >= 1; i-- {
		go h.Broadcast()
	}

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case client := <-h.Register:
			log.Println("Register new machine client in HUB")
			h.rw.Lock()
			_, ok := h.machines[client.ID]
			if ok {
				// log.Println(fmt.Sprintf("Machine with id: %s already exists", client.ID))
				client.HubChan <- false
			} else {
				// log.Println("Welcome new client")
				h.machines[client.ID] = client
				client.HubChan <- true
			}
			h.rw.Unlock()
		case client := <-h.Unregister:
			h.rw.RLock()
			//machine, ok := h.machines[client.ID]
			_, ok := h.machines[client.ID]
			h.rw.RUnlock()
			if ok {
				// log.Println("Unregister machine client")
				// check if registered and unregistering machine ip addresses aare same or not for safety
				// if strings.Compare(client.TConn.RemoteAddr().String(), machine.TConn.RemoteAddr().String()) == 0 {
				h.rw.Lock()
				delete(h.machines, client.ID)
				h.rw.Unlock()
				//}
			}
		case client := <-h.RegisterBot:
			log.Println("Register new Bot client")
			h.rw.Lock()
			_, ok := h.Bots[client.TConn.RemoteAddr().String()]
			if ok {
				client.HubChan <- false
			} else {
				h.Bots[client.TConn.RemoteAddr().String()] = client
				client.HubChan <- true
			}
			h.rw.Unlock()
		case client := <-h.UnRegisterBot:
			log.Println("Unregister Bot client")
			h.rw.RLock()
			_, ok := h.Bots[client.TConn.RemoteAddr().String()]
			h.rw.RUnlock()
			if ok {
				h.rw.Lock()
				delete(h.Bots, client.TConn.RemoteAddr().String())
				h.rw.Unlock()
			}
		case client := <-h.RegisterWs:
			log.Println("Register new Ws client")
			h.rw.Lock()
			_, ok := h.Bots[client.conn.RemoteAddr().String()]
			if ok {
				client.HubChan <- false
			} else {
				h.Bots[client.conn.RemoteAddr().String()] = client
				client.HubChan <- true
			}
			h.rw.Unlock()
		case client := <-h.UnRegisterWs:
			log.Println("Unregister Ws client")
			h.rw.RLock()
			_, ok := h.Bots[client.conn.RemoteAddr().String()]
			h.rw.RUnlock()
			if ok {
				h.rw.Lock()
				delete(h.Bots, client.conn.RemoteAddr().String())
				h.rw.Unlock()
			}
		case <-ticker.C:
			log.Println("Bot/Ws clients online:", len(h.Bots), "Machines online:", len(h.machines), "Routines:", runtime.NumGoroutine())
		}
	}
}

// Broadcast Message brokers
func (h *Hub) Broadcast() {
	// broadcasts message between connected clients (Bots --> machines)
	for {
		select {
		case initializer := <-h.BroadcastChannel:
			h.rw.RLock()

			machine, ok := h.machines[initializer.To]
			if ok {
				machine.Send <- initializer

				init := new(models.Protocol)
				init.To = initializer.To
				init.AdminChan = initializer.AdminChan
				init.Command = "OK"
				init.ErrCode = 1
				init.AdminChan <- init
			} else {
				log.Println(fmt.Sprintf("Error: Client with ID: %s not found", initializer.To))
				initializer.Command = fmt.Sprintf("Client with ID: %s not found", initializer.To)
				initializer.Error = true
				initializer.AdminChan <- initializer
			}
			h.rw.RUnlock()
		}
	}
}

// Pool For controlling memory overhead while multiple websocket connection
func (h *Hub) Pool() {

	workers := os.Getenv("TCP_HUB_WORKERS")
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
