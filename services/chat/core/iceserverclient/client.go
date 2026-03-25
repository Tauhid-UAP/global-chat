package iceserverclient

import (
	"context"
	"log"
	"time"
	"encoding/json"
	"errors"

	"github.com/Tauhid-UAP/global-chat/services/chat/core/redisclient"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/twiliorest"
)

type ICEServerClient struct {
	TwilioClient *twiliorest.TwilioClient
}

func (iceServerClient *ICEServerClient) GetTwilioICEServers() ([] *ICEServer, error) {
	response, err := iceServerClient.TwilioClient.GetTwilioICEServerCredentials()
	if err != nil {
		return nil, err
	}

	twilioIceServers := response.IceServers
	if twilioIceServers == nil {
		return nil, errors.New("No ICE servers returned from Twilio")
	}

	var iceServers []*ICEServer
	for _, server := range *twilioIceServers {
		iceServers = append(iceServers, &ICEServer{
			URLs: []string{server.Urls},
			Username: server.Username,
			Credential: server.Credential,
		})
	}

	return iceServers, nil
}

func (iceServerClient *ICEServerClient) GetICEServersForRoomName(ctx context.Context, roomName string, twilioICEServersTTL time.Duration) ([]*ICEServer, error) {
	cacheKey := "ice_servers:" + roomName
	cachedICEServers, err := redisclient.GetCachedICEServers(ctx, cacheKey)
	if err == nil {
		var iceServers []*ICEServer
		if err := json.Unmarshal([]byte(cachedICEServers), &iceServers); err == nil {
			return iceServers, nil
		}
	}

	iceServers, err := iceServerClient.GetTwilioICEServers()
	if err != nil {
		return nil, err
	}

	marshalledICEServers, err := json.Marshal(iceServers)
	if err != nil {
		log.Printf("Error marshalling ICE servers for caching: %v", err)
		return iceServers, nil
	}

	err = redisclient.SetICEServersToCache(ctx, cacheKey, string(marshalledICEServers), twilioICEServersTTL)
	if err != nil {
		log.Printf("Error setting ICE servers to cache: %v", err)
		return iceServers, nil
	}

	return iceServers, nil
}