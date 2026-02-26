package chat

import (
	"log"
	"context"
	"sync"
	"time"
	"encoding/json"

	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

type Room struct {
	Name string
	Clients map[*Client]struct{}
	Register chan *Client
	Unregister chan *Client
	Done chan struct{}

	mu sync.Mutex
	subscriber func()
}

func (room *Room) Run(ctx context.Context, onEmpty func()) {
	defer onEmpty()
	
	clients := room.Clients

	for {
		select {
			case client := <- room.Register:
				log.Printf("Client - %s joined room %s", client.UserID, client.RoomName)
				room.mu.Lock()
				clients[client] = struct{}{}
				room.mu.Unlock()

			case client := <- room.Unregister:
				roomName := client.RoomName
				log.Printf("Client - %s left room %s", client.UserID, roomName)
				room.mu.Lock()
				delete(clients, client)
				isEmptyRoom := len(clients) == 0
				room.mu.Unlock()

				close(client.Receiver)
				
				userLeftPayload := CreateWebSocketMessageForUserLeaving(client.UserID, client.UserFullName, time.Now().UTC())
				userLeftPayloadBytes, err := json.Marshal(userLeftPayload)
				if err == nil {
					redisclient.PublishToRoom(ctx, roomName, userLeftPayloadBytes)

				} else {
					log.Printf("Error marshalling user left payload: %v", err)
				}

				if isEmptyRoom {
					return
				}

			case <- ctx.Done():
				log.Println("Done")
				return
		}
	}
}

func CreateRoom(name string) *Room {
	return &Room {
		Name: name,
		Clients: make(map[*Client]struct{}),
		Register: make(chan *Client),
		Unregister: make(chan *Client),
		Done: make(chan struct{}),
	}
}
