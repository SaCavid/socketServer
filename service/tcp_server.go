package service

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"socketserver/models"
	ws "socketserver/websocket"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	defaultBuffer = 0
)

type Server struct {
	Port            int
	Mu              sync.Mutex
	DefaultDeadline time.Duration
	Hub             *ws.Hub
}

type User struct {
	Name    string
	Channel chan models.Message
}

// TcpServers - read environment and continue
func (srv *Server) TcpServers() {

	tcpHost := os.Getenv("TCP_HOST_NAME")
	addr := os.Getenv("TCP_PORT")
	defaultSize := os.Getenv("DEFAULT_BUFFER")

	n, err := strconv.ParseInt(defaultSize, 10, 64)
	if err != nil {
		n = 100
	}
	defaultBuffer = int(n)

	ports := strings.Split(addr, ",")

	for _, v := range ports {
		go srv.TcpServer(tcpHost, v)
	}
}

// TcpServer - Will start tcp server per port from environment
func (srv *Server) TcpServer(tcpHost, addr string) {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", tcpHost, addr))
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := l.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	log.Println("Started TCP server on " + addr)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go srv.Receiver(conn)
	}
}

// Receiver - Tcp connection reader
func (srv *Server) Receiver(conn net.Conn) {

	logged := false
	quit := make(chan bool, 1)
	b := make([]byte, defaultBuffer)

	// var user string
	client := &ws.Client{}
	defer func() {
		//		log.Println("TCP connection finished")
		quit <- true
	}()

	for {
		n, err := conn.Read(b)
		if err != nil {
			if err.Error() == "EOF" {
				log.Println("Connection interrupted by client machine")
			} else {
				// log.Println(err)
			}
			srv.Hub.Unregister <- client
			break
		} else {
			b = b[:n]
			if logged {
				// doing nothing if machine logged and sending message
			} else {
				msg := string(b)
				login := strings.Split(msg, ".")

				if len(login) != 2 {
					log.Println("Wrong login msg format")
					break
				}

				// FIXME - check worker name is correct ?

				// FIXME - check access token is correct ?
				client = &ws.Client{
					Id:      fmt.Sprintf("%s", login[1]),
					Send:    make(chan *ws.Protocol, 10),
					TConn:   conn,
					HubChan: make(chan bool, 1),
				}

				go srv.Transmitter(conn, client.Send, quit)
				srv.Hub.Register <- client

				confirm := <-client.HubChan

				if confirm {
					logged = true
					// channel for quitting transmitter goroutine
					//					log.Println("New machine logged:", login[0], " with access key and ID:", login[1])
					close(client.HubChan)
				} else {
					//					log.Println("Error while registering. Closing TCP connection")
					close(client.HubChan)
					break
				}
			}
		}
	}
}

// Transmitter - Tcp connection writer
func (srv *Server) Transmitter(conn net.Conn, c chan *ws.Protocol, quit chan bool) {

	for {
		select {
		case msg := <-c:

			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", msg.Command)))
			if err != nil {
				log.Println(err)
				return
			}

			err = conn.SetDeadline(time.Now().Add(srv.DefaultDeadline))
			if err != nil {
				log.Println(err)
			}
		case b := <-quit:
			if b {
				close(quit)
				close(c)
				_ = conn.Close()
				return
			}
		}
	}
}

// Ws - Websocket endpoint
func (srv *Server) Ws(w http.ResponseWriter, r *http.Request) {
	ws.ServeWs(srv.Hub, w, r)
}
