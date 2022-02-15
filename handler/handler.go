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

func DisplayTitle() {

	// https://patorjk.com/software/taag/#p=display&f=Graffiti&t=Type%20Something%20
	// type something
	fmt.Println(`

___________                        _________                      __  .__    .__                 
\__    ___/__.__.______   ____    /   _____/ ____   _____   _____/  |_|  |__ |__| ____    ____   
  |    | <   |  |\____ \_/ __ \   \_____  \ /  _ \ /     \_/ __ \   __\  |  \|  |/    \  / ___\  
  |    |  \___  ||  |_> >  ___/   /        (  <_> )  Y Y  \  ___/|  | |   Y  \  |   |  \/ /_/  > 
  |____|  / ____||   __/ \___  > /_______  /\____/|__|_|  /\___  >__| |___|  /__|___|  /\___  /  
          \/     |__|        \/          \/             \/     \/          \/        \//_____/   

	`)
}

func Listen() {

	host := os.Getenv("WEB_SOCKET_HOST_NAME")
	httpPort := os.Getenv("WEB_SOCKET_PORT")
	log.Printf("Info: Starting websocket server! --> %s:%s ", host, httpPort)

	srv := &service.Server{}

	go srv.TcpServers()

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
