package chat

import (
	"time"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Receiver chan []byte
	UserID string
	UserFullName string
	RoomName string
}

func (client *Client) ReceiveMessages() {
	log.Printf("Started listening for messages")
	websocketConnection := client.Conn
	defer websocketConnection.Close()
	
	messageType := websocket.TextMessage
	for message := range client.Receiver {
		log.Printf("Client - %s received message: %s", client.UserID, client.RoomName)
		websocketConnection.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := websocketConnection.WriteMessage(messageType, message); err != nil {
			log.Printf("Error writing websocket message for client: %v", err)
			return
		}
	}
}
