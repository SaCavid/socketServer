package service

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	ws "socketserver/websocket"
	"strings"
)

// Server Main struct for controlling servers and etc
type Server struct {
	Hub *ws.Hub
}

// TCPServers - read environment and continue
func (srv *Server) TCPServers() {

	tcpHost := os.Getenv("TCP_HOST_NAME")
	addr := os.Getenv("TCP_PORT")

	ports := strings.Split(addr, ",")

	for _, v := range ports {
		go srv.TCPServer(tcpHost, v)
	}
}

// TCPServer - Will start tcp server per port from environment
func (srv *Server) TCPServer(tcpHost, addr string) {

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
			log.Println(err)
			continue
		}

		go srv.Receiver(conn)
	}
}

// Ws - Websocket endpoint
func (srv *Server) Ws(w http.ResponseWriter, r *http.Request) {
	ws.ServeWs(srv.Hub, w, r)
}
