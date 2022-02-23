package websocket

import (
	"github.com/gorilla/websocket"
	"net"
	"socketserver/models"
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ID string // Client unique id: remote address for unique connection

	Hub *Hub

	// channel to confirm registration
	HubChan chan bool

	// The websocket connection.
	conn *websocket.Conn

	TConn net.Conn
	// Buffered channel of outbound messages.
	Send chan *models.Protocol
}
