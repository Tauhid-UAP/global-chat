package iceserverclient

import (
	"errors"

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

	return iceServers, err
}