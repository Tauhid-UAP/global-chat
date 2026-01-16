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
		
		ctx := context.Background()
		userID := r.Context().Value(middleware.UserIDKey).(string)
		user, _ := store.GetUserByID(ctx, userID)
		userFullName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		
		client := &chat.Client{
			Conn: websocketConnection,
			Receiver: make(chan []byte, 256),
			UserID: userID,
			UserFullName: userFullName,
			RoomName: roomName,
		}

		room := hub.GetOrCreateRoom(ctx, roomName)
		room.Register <- client

		go client.ReceiveMessages()

		userJoinedPayload := chat.CreateWebSocketMessageForUserJoining(userID, userFullName, time.Now().UTC())
		userJoinedPayloadBytes, _ := json.Marshal(userJoinedPayload)
		if err == nil {
			redisclient.PublishToRoom(ctx, roomName, userJoinedPayloadBytes)
		} else {
			log.Printf("Error marshalling user join payload: %v", err)
		}


		for {
			_, message, err := websocketConnection.ReadMessage()
			log.Printf("Message: %s", string(message))
			if err != nil {
				log.Printf("Error reading websocket message: %v", err)
				break
			}

			/*
			payload := chat.WebSocketMessage{
				Type: chat.EventChatMessage,
				Data: chat.ChatMessageData{
					FullName: fmt.Sprintf("%s %s", user.FirstName, user.LastName),
					Message: string(message),
					SentAt: time.Now().UTC(),
				},
			}
			*/
			payload := chat.CreateWebSocketMessageForChatMessageData(
				userID,
				userFullName,
				string(message),
				time.Now().UTC(),
			)
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
