package service

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"socketserver/models"
	ws "socketserver/websocket"
	"sync"
	"time"
)

func (srv *Server) TcpServer(addr string) {

	l, err := net.Listen("tcp", ":"+addr)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := l.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	log.Println("started server on " + addr)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		err = conn.SetReadDeadline(time.Now().Add(time.Second * 120))
		if err != nil {
			log.Println(err)
			return
		}

		go srv.Receiver(conn)
	}
}

type Server struct {
	Port               int
	Mu                 sync.Mutex
	LoginChan          chan *User
	LogoutChan         chan string
	Clients            map[string]chan models.Message
	ReceiverRoutine    uint64
	TransmitterRoutine uint64
	SendMessages       uint64
	ReceivedMessages   uint64
	DefaultDeadline    time.Duration
	Hub                *ws.Hub
}

type User struct {
	Name    string
	Channel chan models.Message
}

func (srv *Server) Receiver(conn net.Conn) {

	logged := false
	// var user string

	c := make(chan models.Message, 8)
	srv.ReceiverRoutine++
	defer func() {

		_ = conn.Close()

		srv.ReceiverRoutine--
	}()

	go srv.Transmitter(conn, c)

	for {
		m := models.Message{}
		d := json.NewDecoder(conn)

		err := d.Decode(&m)
		if err != nil {
			log.Println(err)
			return
		}

		err = m.ValidateMessage()
		if err != nil {
			log.Println(err)
			continue
		}

		if !logged {
			logged = true
			// user = m.From
			//srv.Login(user, c)
		} else {
			srv.Mu.Lock()

			receiver := srv.Clients[m.To]
			srv.ReceivedMessages++
			srv.Mu.Unlock()

			if receiver != nil {
				receiver <- m
			}

			err = conn.SetReadDeadline(time.Now().Add(srv.DefaultDeadline))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func (srv *Server) Transmitter(conn net.Conn, c chan models.Message) {

	srv.TransmitterRoutine++
	defer func() {
		close(c)
		srv.TransmitterRoutine--
		_ = conn.Close()
	}()

	for {
		y := <-c

		if y.Status {
			return
		}

		srv.Mu.Lock()
		srv.SendMessages++
		srv.Mu.Unlock()
		d, err := json.Marshal(y)
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = conn.Write(d)
		if err != nil {
			log.Println(err)
			return
		}

		err = conn.SetDeadline(time.Now().Add(srv.DefaultDeadline))
		if err != nil {
			log.Println(err)
		}
	}
}

func (srv *Server) Ws(w http.ResponseWriter, r *http.Request) {
	ws.ServeWs(srv.Hub, w, r)
}
