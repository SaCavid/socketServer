package service

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"socketserver/models"
	ws "socketserver/websocket"
	"strconv"
	"strings"
	"time"
)

// Receiver - Tcp connection reader
func (srv *Server) Receiver(conn net.Conn) {

	logged := false
	send := make(chan *models.Protocol, 10)
	quit := make(chan bool, 1)

	defaultSize := os.Getenv("DEFAULT_BUFFER")

	n, err := strconv.ParseInt(defaultSize, 10, 64)
	if err != nil {
		n = 100
	}
	defaultBuffer := int(n)

	b := make([]byte, defaultBuffer)

	machine := &ws.Client{}
	defer func() {
		//		log.Println("TCP connection finished")
		quit <- true
	}()
	t := time.Now()
	for {

		if !logged {
			// 60 seconds for login
			// conn will be interrupted if not login happened
			if time.Since(t).Seconds() > 60 {
				log.Println("Login attempt failed")
				return
			}
		}

		// n = count of bytes read from port
		n, err := conn.Read(b)
		if err != nil {
			if err.Error() == "EOF" {
				// log.Println("Connection interrupted by client machine")
			} else {
				// log.Println(err)
			}
			srv.Hub.Unregister <- machine
			break
		} else {
			b = b[:n]
			if logged {
				// doing nothing if machine logged and sending message to server. It's not allowed by project
			} else {

				go srv.Transmitter(conn, send, quit)

				login, err := ValidateLogin(b)
				if err != nil {
					log.Println(err)
					return
				}

				// create machine client
				machine = &ws.Client{
					ID:      fmt.Sprintf("%s", login[1]),
					Send:    send,
					TConn:   conn,
					HubChan: make(chan bool, 1),
				}

				// register machine client to HUB
				srv.Hub.Register <- machine

				// get confirmation about registration from HUB
				confirm := <-machine.HubChan

				if confirm {
					logged = true
					close(machine.HubChan)
				} else {
					close(machine.HubChan)
					break
				}
			}
		}
	}
}

// ValidateLogin checks first message from machine for valid login information
func ValidateLogin(b []byte) ([]string, error) {
	if len(b) > 30 {
		return nil, fmt.Errorf("wrong message format: max 30 characters allowed")
	}

	msg := string(b)
	login := strings.Split(msg, ".")

	if len(login) != 2 {
		return nil, fmt.Errorf("wrong login msg format")
	}

	re := regexp.MustCompile("^[a-zA-Z0-9]")
	if !re.MatchString(login[0]) {
		return nil, fmt.Errorf("wrong access key format")
	}

	match, _ := regexp.MatchString("Worker([0-9]+)", login[1])
	if !match {
		return nil, fmt.Errorf("not correct machine name")
	}

	return login, nil
}
