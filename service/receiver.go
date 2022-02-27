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

	// allowed time for idle connection
	var timeout float64 = 10

	// channel for safe quit of transmitter and receiver routines
	quit := make(chan bool, 1)

	// default buffer size = 60
	b := make([]byte, 60)

	// create machine client
	machine := &ws.Client{
		TConn:   conn,
		Hub:     srv.Hub,
		HubChan: make(chan bool, 1),
		Send:    make(chan *models.Protocol, 10),
	}

	defer func() {
		// log.Println("TCP connection finished")
		if logged {
			srv.Hub.Unregister <- machine
			quit <- true
		} else {
			_ = conn.Close()
			close(quit)
			close(machine.Send)
		}

		// log.Println("Quit normally")
	}()

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
				// log.Println("Connection interrupted by client machine")
			} else {
				// log.Println(err)
			}

			if !logged {
				// if client not logged and sending malformed message close all channels and quit receiver goroutine
				// Invalid command
				log.Println("Invalid worker command")
				_, _ = conn.Write([]byte(models.InvalidWorkerCommand.String() + " \n"))

				return
			}
			break
		} else if !logged {
			b = b[:n]

			login, bot, Msg, err := ValidateLogin(b)
			if err != nil {
				log.Println(err)
				_, _ = conn.Write([]byte(Msg.String() + "\n"))
				break
			}

			// update machine client ID - access-token.workerName
			machine.ID = fmt.Sprintf("%s.%s", login[0], login[1])

			if bot {
				ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

				if !strings.Contains(ip, "127.0.") && !strings.Contains(ip, "10.10.") { //&& !strings.Contains(ip, "192.168.") {
					msg := models.Unauthorized
					_, err := conn.Write([]byte(msg.String() + "\n"))
					if err != nil {
						log.Println(err)
					}
					return
				} else {
					b = b[:n]
					command, msg, err := ParseCommand(b)
					if err != nil {
						log.Println(err)
						_, _ = conn.Write([]byte(msg.String() + "\n"))
						break
					}

					command.AdminChan = machine.Send
					srv.Hub.BroadcastChannel <- command

					receive := <-command.AdminChan

					_, _ = conn.Write([]byte(receive.Command + "\n"))
					return
				}
			} else {

				// max 60 seconds for confirmation
				ticker := time.NewTicker(time.Duration(timeout) * time.Second)

				// register client to HUB
				srv.Hub.Register <- machine

				// get confirmation about registration from HUB
				for {
					select {
					// get register confirmation from hub
					case confirm := <-machine.HubChan:

						if confirm {
							logged = true
							ticker.Stop()
							close(machine.HubChan)

							// Login message for client(machine)
							go srv.Transmitter(conn, machine.Send, quit)

							// send Welcome message to connection
							machine.Send <- &models.Protocol{
								Command: models.Welcome.String(),
							}

							break
						} else {
							_, _ = conn.Write([]byte(models.Reserved.String() + " \n"))
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
			}
		}
	}
}

// ValidateLogin checks first message from machine for valid login information. return true if its bot client
func ValidateLogin(b []byte) ([]string, bool, models.Msg, error) {
	bot := false
	if len(b) > 30 {
		return nil, false, models.MalformedCommand, fmt.Errorf("wrong message format: max 30 characters allowed")
	}

	msg := string(b)
	login := strings.Split(msg, ".")

	if strings.Contains(msg, "|") {
		bot = true
	}

	if len(login) != 2 {
		return nil, false, models.InvalidWorker, fmt.Errorf("wrong login msg format")
	}

	msgErr, err := ValidateAccessToken(login[0])
	if err != nil {
		return nil, false, msgErr, err
	}

	worker := strings.Split(login[1], "|")
	msgErr, err = ValidateWorkerName(worker[0])
	if err != nil {
		return nil, false, msgErr, err
	}

	return login, bot, models.Ok, nil
}

// ParseCommand checks if command send by bot to server is correct format or not
func ParseCommand(b []byte) (*models.Protocol, models.Msg, error) {

	if len(b) > 30 {
		return nil, models.Failed, fmt.Errorf("wrong command message format: max 30 characters allowed")
	}

	msg := string(b)

	if !strings.ContainsAny(msg, ".|") {
		return nil, models.MalformedCommand, fmt.Errorf("malformed command")
	}

	command := strings.Split(msg, ".")
	str := strings.Split(command[1], "|")

	if len(str) != 2 {
		return nil, models.Failed, fmt.Errorf("wrong command format: no command found")
	}

	command[1] = str[0]
	command = append(command, str[1])

	// Example: s0m4k3y.Worker001|SHUTDOWN
	if len(command) != 3 {
		return nil, models.Failed, fmt.Errorf("wrong command message format")
	}
	cmd := &models.Protocol{
		To:      fmt.Sprintf("%s.%s", command[0], command[1]),
		Command: command[2],
	}

	validate, err := cmd.Validate()
	if err != nil {
		return nil, validate, err
	}

	return cmd, models.Ok, nil
}

func ValidateAccessToken(token string) (models.Msg, error) {

	// Example token: s0m4k3y
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	match := re.MatchString(token)
	if !match {
		return models.InvalidWorker, fmt.Errorf("wrong access key format")
	}

	return models.Ok, nil
}

func ValidateWorkerName(workerName string) (models.Msg, error) {

	// Example workerName: Worker001
	re := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	match := re.MatchString(workerName)
	if !match {
		return models.InvalidWorker, fmt.Errorf("wrong worker name format")
	}

	return models.Ok, nil
}
