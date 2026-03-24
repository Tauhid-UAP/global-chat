package iceserverclient

type ICEServer struct {
	URLs []string `json:"urls"`
	Username string `json:"username,omitempty"`
	Credential string `json:"credential,omitempty"`
}