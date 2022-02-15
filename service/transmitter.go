package service

import (
	"fmt"
	"log"
	"net"
	"socketserver/models"
)

// Transmitter - Tcp connection writer
func (srv *Server) Transmitter(conn net.Conn, c chan *models.Protocol, quit chan bool) {

	for {
		select {
		case msg := <-c:

			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", msg.Command)))
			if err != nil {
				log.Println(err)
				return
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
