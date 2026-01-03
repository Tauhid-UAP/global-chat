package chat

import (
	"context"
	"sync"
	"log"

	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

type Hub struct {
	mu sync.RWMutex
	rooms map[string]*Room
}

func (hub *Hub) GetOrCreateRoom(ctx context.Context, name string) *Room {
	rooms := hub.rooms
	/* Fast path for high read volumes */
	hub.mu.RLock()
	if room, ok := rooms[name]; ok {
		log.Println("Got room")
		hub.mu.RUnlock()
		return room
	}
	hub.mu.RUnlock()

	/* Slow path in case the room doesn't exist, until it is initialized. */
	hub.mu.Lock()
	defer hub.mu.Unlock()
	
	/*
	Perform redundant check in the slow path.
	Since multiple threads might have been contending for this lock, one may have already initiated the room.
	So, subsequent holders of the slow path lock should double-check if it has already been initialized.
	*/
	if room, ok := rooms[name]; ok {
		log.Println("Got room")
		return room
	}

	room := CreateRoom(name)
	rooms[name] = room

	redisPubSubSubscription := redisclient.SubscribeToRoom(ctx, name)
	subscribedChannel := redisPubSubSubscription.Channel()

	go room.Run(ctx, func() {
		log.Println("Closing Redis subscription")
		redisPubSubSubscription.Close()
		hub.mu.Lock()
		delete(rooms, name)
		hub.mu.Unlock()
	})

	go func() {
		for message := range subscribedChannel {
			room.mu.Lock()
			payload := message.Payload
			log.Printf("Broadcasting message to clients: %s", payload)
			for client := range room.Clients {
				select {
					case client.Receiver <- []byte(payload):
					default:
				}
			}
			room.mu.Unlock()
		}

		log.Println("How did I get here?")
	}()

	return room
}

func CreateHub() *Hub {
	return &Hub {rooms: make(map[string]*Room)}
}
