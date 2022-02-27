package handler

import (
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"os"
	"socketserver/service"
	"socketserver/websocket"
	"time"
)

// Listen start listening services: WebSocket Server, TCP Server
func Listen() {

	host := os.Getenv("WEB_SOCKET_HOST_NAME")
	httpPort := os.Getenv("WEB_SOCKET_PORT")
	log.Printf("Info: Starting websocket server! --> %s:%s ", host, httpPort)

	srv := &service.Server{}

	go srv.TCPServers()

	hub := websocket.NewHub()
	go hub.Pool()
	go hub.Run()
	srv.Hub = hub

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", Index).Methods("GET")
	r.HandleFunc("/ws", srv.Ws)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", host, httpPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 50 * time.Second,
		IdleTimeout:  time.Second * 100,
	}
	/*
	   FIXME Question: Are your machines after successful login need to send message to server ? - At the moment we have 2 goroutines for receive and for transmit of message. If your machines will not send anything at any way after login to server I can skip receive goroutine which will reduce recourses used by server for machines for 50%. In this case Server will be able to forward commands to machines only. On close of connection by server or client both server and I guess your machines will understand that connection is closed so we will not get error on server.
	*/
	log.Fatal(server.ListenAndServe())
}

// Index simple test page
func Index(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.New("index.html").ParseFiles("./assets/index.html")
	if err != nil {
		log.Println(err.Error())
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		panic(err)
	}
}
