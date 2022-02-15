package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
	"socketserver/service"
	"socketserver/websocket"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	srv := &service.Server{
		Port:            0,
		DefaultDeadline: time.Second * 120,
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

	go srv.TcpServers()
	r.HandleFunc("/", Index).Methods("GET")
	r.HandleFunc("/ws", srv.Ws)

	host := os.Getenv("WEB_SOCKET_HOST_NAME")
	httpPort := os.Getenv("WEB_SOCKET_PORT")
	log.Printf("Info: Starting websocket server! --> %s:%s ", host, httpPort)
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", host, httpPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 50 * time.Second,
		IdleTimeout:  time.Second * 100,
	}

	log.Fatal(server.ListenAndServe())
}

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
