package service

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"socketserver/models"
	ws "socketserver/websocket"
	"strings"
	"time"
)

// Receiver - Tcp connection reader
func (srv *Server) Receiver(conn net.Conn) {

	logged := false
	bot := false
	var timeout float64 = 10
	// channel for safe quit of transmitter and receiver routines
	quit := make(chan bool, 1)

	// default buffer size = 60
	b := make([]byte, 60)

	// create machine client
	machine := &ws.Client{
		Send:    make(chan *models.Protocol, 10),
		TConn:   conn,
		Hub:     srv.Hub,
		HubChan: make(chan bool, 1),
	}

	defer func() {
		// log.Println("TCP connection finished")
		if logged {
			if bot {
				srv.Hub.UnRegisterBot <- machine
			} else {
				srv.Hub.Unregister <- machine
			}
		}

		quit <- true
		// log.Println("Quit normally")
	}()

	// Login message for client(machine)
	go srv.Transmitter(conn, machine.Send, quit)
	t := time.Now()
	for {

		if !logged {
			// 60 seconds for login
			// conn will be interrupted if not login happened
			if time.Since(t).Seconds() > timeout {
				log.Println("Login attempt failed: timeout")
				return
			}
		}

		// n = count of bytes read from port
		n, err := conn.Read(b)
		if err != nil {
			if err.Error() == "EOF" {
				log.Println("Connection interrupted by client machine")
			} else {
				log.Println(err)
			}

			if !logged {
				// if client not logged and sending malformed message close all channels and quit receiver goroutine
				msg := "Invalid command\n"
				_, err := conn.Write([]byte(msg))
				if err != nil {
					log.Println(err)
				}
				return
			}
			break
		} else if !logged {
			b = b[:n]
			// log.Println(string(b), logged)

			login, bt, err := ValidateLogin(b)
			if err != nil {
				log.Println(err)
				break
			}

			// update machine client
			machine.ID = fmt.Sprintf("%s", login[1])

			// max 60 seconds for confirmation
			ticker := time.NewTicker(time.Duration(timeout) * time.Second)

			// register client to HUB
			if bt {
				srv.Hub.RegisterBot <- machine
				bot = bt
			} else {
				srv.Hub.Register <- machine
			}

			// get confirmation about registration from HUB
			for {
				select {
				// get register confirmation from hub
				case confirm := <-machine.HubChan:

					if confirm {
						logged = true
						ticker.Stop()
						close(machine.HubChan)

						// send Welcome message to connection
						_, err := conn.Write([]byte("WELCOME\n"))
						if err != nil {
							log.Println(err)
							return
						}

						//if bot {
						//	log.Println("Logged new client(bot)", login)
						//} else {
						//	log.Println("Logged new client(machine)", login)
						//}
						break
					} else {
						// log.Println("Can't register new client")
						return
					}

				// after 60 seconds close idle connection
				case _ = <-ticker.C:
					log.Println("Login attempt failed: timeout")
					ticker.Stop()
					break
				}
				if logged {
					break
				}
			}

			log.Println("User logged")
		}

		// receive messages and broadcast after login if its bot client
		if bot {
			b = b[:n]
			command, err := ParseCommand(b)
			if err != nil {
				log.Println(err)
				break
			}

			err = command.Validate()
			if err != nil {
				log.Println(err)
				break
			}

			log.Println(string(b))
			command.AdminChan = machine.Send
			srv.Hub.BroadcastChannel <- command
		}
	}
}

// ValidateLogin checks first message from machine for valid login information. return true if its bot client
func ValidateLogin(b []byte) ([]string, bool, error) {
	bot := false

	if len(b) > 30 {
		return nil, false, fmt.Errorf("wrong message format: max 30 characters allowed")
	}

	msg := string(b)
	login := strings.Split(msg, ".")

	if strings.Contains(msg, "|") {
		bot = true
	}

	if len(login) != 2 {
		return nil, bot, fmt.Errorf("wrong login msg format")
	}

	// Example token: s0m4k3y
	if strings.ContainsAny(login[0], " ") {
		log.Println("Access token cant have whitespaces")
		return nil, bot, fmt.Errorf("access token error")
	}

	// Example machine name: Worker001
	if strings.ContainsAny(login[1], " ") {
		return nil, bot, fmt.Errorf("machine name cant have whitespaces")
	}

	re := regexp.MustCompile("^[a-zA-Z0-9]")
	if !re.MatchString(login[0]) {
		return nil, bot, fmt.Errorf("wrong access key format")
	}

	match, _ := regexp.MatchString("Worker([0-9]+)", login[1])
	if !match {
		return nil, bot, fmt.Errorf("not correct machine name")
	}

	return login, bot, nil
}

// ParseCommand checks if command send by bot to server is correct format or not
func ParseCommand(b []byte) (*models.Protocol, error) {

	if len(b) > 30 {
		return nil, fmt.Errorf("wrong command message format: max 30 characters allowed")
	}

	msg := string(b)

	if !strings.ContainsAny(msg, ".|") {
		return nil, fmt.Errorf("malformed command")
	}

	command := strings.Split(msg, ".")
	str := strings.Split(command[1], "|")

	if len(str) != 2 {
		return nil, fmt.Errorf("wrong command format: no command found")
	}

	command[1] = str[0]
	command = append(command, str[1])

	// Example: s0m4k3y.Worker001|SHUTDOWN
	if len(command) != 3 {
		return nil, fmt.Errorf("wrong command message format")
	}

	// Example token: s0m4k3y
	if strings.ContainsAny(command[0], " ") {
		log.Println("Access token cant have whitespaces")
		return nil, fmt.Errorf("access token error")
	}

	// Example machine name: Worker001
	if strings.ContainsAny(command[1], " ") {
		return nil, fmt.Errorf("machine name cant have whitespaces")
	}

	re := regexp.MustCompile("^[a-zA-Z0-9]")
	if !re.MatchString(command[0]) {
		return nil, fmt.Errorf("wrong access key format")
	}

	match, _ := regexp.MatchString("Worker([0-9]+)", command[1])
	if !match {
		return nil, fmt.Errorf("not correct machine name")
	}

	return &models.Protocol{
		To:      command[1],
		Command: command[2],
	}, nil
}
