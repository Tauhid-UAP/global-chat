package websockethandlers

import (
	"context"
	"net/http"
	"log"
	"fmt"
	"time"
	"encoding/json"

	"github.com/gorilla/websocket"

	"github.com/Tauhid-UAP/global-chat/core/chat"
	"github.com/Tauhid-UAP/global-chat/core/middleware"
	"github.com/Tauhid-UAP/global-chat/core/redisclient"
	"github.com/Tauhid-UAP/global-chat/core/store"
)

func ChatHandler(websocketUpgrader websocket.Upgrader, hub *chat.Hub) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		roomName := r.URL.Query().Get("roomName")
		if roomName == "" {
			http.Error(w, "roomName required", http.StatusBadRequest)
			return
		}

		websocketConnection, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		userID := r.Context().Value(middleware.UserIDKey).(string)
		
		client := &chat.Client{
			Conn: websocketConnection,
			Receiver: make(chan []byte, 256),
			UserID: userID,
			RoomName: roomName,
		}

		ctx := context.Background()
		room := hub.GetOrCreateRoom(ctx, roomName)
		room.Register <- client

		go client.ReceiveMessages()
		
		user, _ := store.GetUserByID(ctx, userID)

		for {
			_, message, err := websocketConnection.ReadMessage()
			log.Printf("Message: %s", string(message))
			if err != nil {
				log.Printf("Error reading websocket message: %v", err)
				break
			}

			// payload := []byte(userID + ": " + string(message))
			payload := chat.WebSocketMessage{
				Type: chat.EventChatMessage,
				Data: chat.ChatMessageData{
					FullName: fmt.Sprintf("%s %s", user.FirstName, user.LastName),
					Message: string(message),
					SentAt: time.Now().UTC(),
				},
			}
			log.Printf("Publishing to room: %s", payload)

			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				log.Printf("Error marshalling payload: %v", err)
				continue
			}

			redisclient.PublishToRoom(ctx, roomName, payloadBytes)
		}
		
		log.Printf("Unregistering client")
		room.Unregister <- client
	}
}
