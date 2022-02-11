package main

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"socketserver/service"
	"socketserver/websocket"
	"sync"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)
}

func main() {
	// initialize server controller
	srv := service.Server{
		Port:               0,
		Mu:                 sync.Mutex{},
		LoginChan:          nil,
		LogoutChan:         nil,
		Clients:            nil,
		ReceiverRoutine:    0,
		TransmitterRoutine: 0,
		SendMessages:       0,
		ReceivedMessages:   0,
		DefaultDeadline:    0,
		Hub:                nil,
	}
	// start hub

	// start websocket workers

	// start tcp workers

	// initialize tcp server

	// start websocket http server
	r := mux.NewRouter().StrictSlash(true)
	hub := websocket.NewHub()
	go hub.Pool()
	go hub.Run()
	srv.Hub = hub

	r.HandleFunc("/", Chat).Methods("GET")

	r.HandleFunc("/", srv.Ws).Methods("POST")

	log.Println("Info: Starting websocket server! ")
	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 50 * time.Second,
		IdleTimeout:  time.Second * 100,
	}

	//Key and cert are coming from Let's Encrypt
	log.Fatal(server.ListenAndServe())
}
func Chat(w http.ResponseWriter, _ *http.Request) {
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
