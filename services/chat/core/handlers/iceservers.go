package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"fmt"

	"github.com/Tauhid-UAP/global-chat/services/chat/core/middleware"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/chat"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/iceserverclient"
)

func ICEServersHandler(
	hub *chat.Hub,
	iceServerClient *iceserverclient.ICEServerClient,
	twilioICEServersTTL time.Duration,
) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		roomName := r.URL.Query().Get("room")
		if roomName == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		room := hub.GetRoom(roomName)
		if room == nil {
			http.Error(w, fmt.Sprintf("Invalid room name: %s", roomName), http.StatusUnprocessableEntity)
			return
		}

		ctx := r.Context()
		userID := ctx.Value(middleware.UserIDKey).(string)
		client := room.GetClientWithUserID(userID)
		if client == nil {
			http.Error(w, fmt.Sprintf("You are not in room - %s", userID, roomName), http.StatusForbidden)
			return
		}

		iceServers, err := iceServerClient.GetICEServersForRoomName(ctx, roomName, twilioICEServersTTL)
		if err != nil {
			log.Printf("Failed to fetch ICE servers: %v", err)
			http.Error(w, "Failed to fetch ICE servers.", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"iceServers": iceServers,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}