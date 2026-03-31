package websockethandlers

import (
	"context"
	"net/http"
	"log"
	"fmt"
	"time"
	"encoding/json"

	"github.com/gorilla/websocket"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"

	"github.com/Tauhid-UAP/global-chat/services/chat/core/chat"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/middleware"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/redisclient"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/userselector"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/sfuclient"
)

func ChatHandler(
	websocketUpgrader websocket.Upgrader,
	hub *chat.Hub,
	sfuClient *sfuclient.SFUClient,
	websocketDurationControlConfig *chat.WebsocketDurationControlConfig,
) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		roomName := r.URL.Query().Get("roomName")
		if roomName == "" {
			http.Error(w, "roomName required", http.StatusBadRequest)
			log.Printf("No room name")
			return
		}

		log.Printf("Upgrading connection")

		websocketConnection, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Connection failed")
			return
		}

		log.Printf("Connection upgraded")

		pongDeadline := websocketDurationControlConfig.PongDeadline
		websocketConnection.SetReadDeadline(time.Now().Add(pongDeadline))
		
		ctx := context.Background()
		userID := r.Context().Value(middleware.UserIDKey).(string)

		websocketConnection.SetPongHandler(func(string) error {
			log.Printf("PONG received - %s\n", userID)
			websocketConnection.SetReadDeadline(time.Now().Add(pongDeadline))
			return nil
		})

		isAnonymousUser := r.Context().Value(middleware.IsAnonymousUserKey).(bool)
		user, _ := userselector.GetUserByIDFromApplicableStore(ctx, userID, isAnonymousUser)
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

		go client.ReceiveMessages(websocketDurationControlConfig.PingInterval, websocketDurationControlConfig.WriteDeadline)

		userSignalStream, err := sfuClient.CreateUserStream()
		if err != nil {
			log.Println("Failed to create gRPC stream: ", err)
			return
		}
		client.SFUStream = userSignalStream

		go handleSFUResponses(client)

		userJoinedPayload := chat.CreateWebSocketMessageForUserJoining(userID, userFullName, time.Now().UTC())
		userJoinedPayloadBytes, _ := json.Marshal(userJoinedPayload)
		if err == nil {
			redisclient.PublishToRoom(ctx, roomName, userJoinedPayloadBytes)
		} else {
			log.Printf("Error marshalling user join payload: %v", err)
		}


		for {
			_, messageBytes, err := websocketConnection.ReadMessage()
			log.Printf("messageBytes: %s", string(messageBytes))
			if err != nil {
				log.Printf("Error reading websocket message: %v", err)
				break
			}

			var requestWebsocketMessage chat.RequestWebSocketMessage
			if err := json.Unmarshal(messageBytes, &requestWebsocketMessage); err != nil {
				log.Printf("Error unmarshalling websocket message: %v", err)
				continue
			}

			switch requestWebsocketMessage.Type {

			case chat.EventChatMessage:

				var chatData map[string]string
				if err := json.Unmarshal(requestWebsocketMessage.Data, &chatData); err != nil {
					log.Printf("Error unmarshalling chat data: %v", err)
					continue
				}
				payload := chat.CreateWebSocketMessageForChatMessageData(
					userID,
					userFullName,
					chatData["Message"],
					time.Now().UTC(),
				)
				log.Printf("Publishing to room: %s", payload)

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					log.Printf("Error marshalling payload: %v", err)
					continue
				}

				redisclient.PublishToRoom(ctx, roomName, payloadBytes)
			case chat.EventWebRTCOffer:
				var offer chat.OfferPayload
				if err := json.Unmarshal(requestWebsocketMessage.Data, &offer); err != nil {
					log.Printf("Error unmarshalling WebRTC Offer: %v", err)
					break
				}

				req := &sfupb.SignalRequest{
					RoomName: roomName,
					UserId: userID,
					Payload: &sfupb.SignalRequest_Offer{
						Offer: &sfupb.WebRTCOffer{
							Sdp: offer.SDP,
						},
					},
				}

				client.SFUStream.Send(req)

			case chat.EventWebRTCICECandidate:
				var iceCandidate sfupb.WebRTCICECandidate
				if err := json.Unmarshal(requestWebsocketMessage.Data, &iceCandidate); err != nil {
					log.Printf("Error unmarshalling ICE Candidate: &v", err)
					break
				}

				req := &sfupb.SignalRequest{
					RoomName: roomName,
					UserId: userID,
					Payload: &sfupb.SignalRequest_IceCandidate{
						IceCandidate: &iceCandidate,
					},
				}

				client.SFUStream.Send(req)

			case chat.EventWebRTCPeerLeft:
				client.SFUStream.Close()
			}
		}
		
		client.SFUStream.Close()
		log.Printf("Unregistering client")
		room.Unregister <- client
	}
}

func handleSFUResponses(client *chat.Client) {
	stream := client.SFUStream.Stream
	
	for {
		res, err := stream.Recv()
		if err != nil {
			return
		}

		switch payload := res.Payload.(type) {

		case *sfupb.SignalResponse_Answer:
			answerMessage := map[string]interface{}{
				"Type": chat.EventWebRTCAnswer,
				"Data": map[string]string{
					"sdp": payload.Answer.Sdp,
				},
			}

			bytesMessage, _ := json.Marshal(answerMessage)
			client.Receiver <- bytesMessage
			
		case *sfupb.SignalResponse_IceCandidate:
			iceCandidate := payload.IceCandidate
			iceCandidateMessage := map[string]interface{}{
				"Type": chat.EventWebRTCICECandidate,
				"Data": map[string]interface{}{
					"candidate": iceCandidate.Candidate,
					"sdpMid": iceCandidate.SdpMid,
					"sdpMLineIndex": iceCandidate.SdpMlineIndex,
				},
			}

			bytesMessage, _ := json.Marshal(iceCandidateMessage)
			client.Receiver <- bytesMessage
			
		}
	}
}
