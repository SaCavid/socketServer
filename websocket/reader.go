package websocket

import (
	"github.com/gorilla/websocket"
	"log"
	"os"
	"socketserver/models"
	"strconv"
)

// readPump pumps messages from the websocket connection to the Hub.
func (c *Client) readPump() {
	quit := make(chan bool, 1)
	defer func() {
		quit <- true
		log.Println("Stopped read pump")
	}()

	go c.writePump(quit)

	// Maximum message size allowed from peer/websocket.
	maxSize := os.Getenv("MAX_WEBSOCKET_MESSAGE_SIZE")

	maxMessageSize, err := strconv.ParseInt(maxSize, 10, 64)
	if err != nil {
		maxMessageSize = 512
	}

	c.conn.SetReadLimit(maxMessageSize)

	for {

		msg := &models.Protocol{}

		err := c.conn.ReadJSON(msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			log.Println(err)
			break
		} else {

			err = msg.Validate()
			if err != nil {
				log.Println(err)
				c.Send <- &models.Protocol{
					Error:   true,
					ErrCode: 0,
					Msg:     err.Error(),
				}
			} else {
				msg.AdminChan = c.Send
				c.Hub.BroadcastChannel <- msg
			}
		}
	}
}
