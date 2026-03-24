package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Tauhid-UAP/global-chat/services/chat/core/iceserverclient"
)

func ICEServersHandler(
	iceServerClient *iceserverclient.ICEServerClient,
) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		iceServers, err := iceServerClient.GetTwilioICEServers()
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