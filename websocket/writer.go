package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"socketserver/models"
)

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	defer func() {
		log.Println("Stopped write pump")
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// The hub closed the channel.
				log.Println("Closed channel:")
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println(err)
				return
			}

			text, err := json.Marshal(&models.Response{
				To:      message.To,
				Command: message.Command,
				Error:   message.Error,
				ErrCode: message.ErrCode,
				Msg:     message.Msg,
			})

			if err != nil {
				log.Println(err)
				return
			}

			_, _ = w.Write(text)
			if err := w.Close(); err != nil {
				return
			}

		}
	}
}
