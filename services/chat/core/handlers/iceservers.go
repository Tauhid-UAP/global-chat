package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Tauhid-UAP/global-chat/services/chat/core/iceserverclient"
)

func ICEServersHandler(
	iceServerClient *iceserverclient.ICEServerClient,
	twilioICEServersTTL time.Duration,
) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		roomName := r.URL.Query().Get("room")
		if roomName == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
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