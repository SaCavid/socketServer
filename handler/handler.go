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
	r.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	//r.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	//r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	//r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	//r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	//r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", host, httpPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 50 * time.Second,
		IdleTimeout:  time.Second * 100,
	}

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
