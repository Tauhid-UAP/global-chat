package twiliorest

import (
	twilio "github.com/twilio/twilio-go"
	twilio2010 "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioClient struct {
	RestClient *twilio.RestClient
}

func (tc *TwilioClient) GetTwilioICEServerCredentials() (*twilio2010.ApiV2010Token, error) {
	params := &twilio2010.CreateTokenParams{}
	response, err := tc.RestClient.Api.CreateToken(params)
	return response, err
}

func CreateTwilioRestClient(accountSID, authToken string) *twilio.RestClient {
	return twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
}