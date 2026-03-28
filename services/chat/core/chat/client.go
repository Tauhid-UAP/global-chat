package chat

import (
	"time"
	"log"

	"github.com/gorilla/websocket"

	"github.com/Tauhid-UAP/global-chat/services/chat/core/sfuclient"
)

type Client struct {
	Conn *websocket.Conn
	Receiver chan []byte
	UserID string
	UserFullName string
	RoomName string

	SFUStream *sfuclient.UserSignalStream
}

func (client *Client) ReceiveMessages(pingInterval, writeDeadline time.Duration) {
	log.Printf("Started listening for messages")
	websocketConnection := client.Conn
	defer websocketConnection.Close()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	userID := client.UserID
	for {
		select {
		case message, ok := <- client.Receiver:
			log.Printf("Client - %s received message: %s", userID, client.RoomName)

			websocketConnection.SetWriteDeadline(time.Now().Add(writeDeadline))
			if !ok {
				websocketConnection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if err := websocketConnection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing websocket message for client: %v", err)
				return
			}

		case <- ticker.C:
			websocketConnection.SetWriteDeadline(time.Now().Add(writeDeadline))

			if err := websocketConnection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error: %v\n", err)
				return
			}

			log.Printf("PING sent - %s\n", userID)
		}
	}
	
	// messageType := websocket.TextMessage
	// for message := range client.Receiver {
	// 	log.Printf("Client - %s received message: %s", client.UserID, client.RoomName)
	// 	websocketConnection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	// 	if err := websocketConnection.WriteMessage(messageType, message); err != nil {
	// 		log.Printf("Error writing websocket message for client: %v", err)
	// 		return
	// 	}
	// }
}

