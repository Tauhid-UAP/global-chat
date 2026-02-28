package sfuserver

import (
	"fmt"
	"log"
	"io"
	"sync"

	"github.com/pion/webrtc/v3"
	
	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/peer"
	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/room"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

// SFUServer implements the gRPC SFUService
type SFUServer struct {
	sfupb.UnimplementedSFUServiceServer
	Rooms map[string]*room.Room
	mu sync.RWMutex
}

func (s *SFUServer) getOrCreateRoom(roomName string) *room.Room {
	rooms := s.Rooms
	s.mu.RLock()
	if fetchedRoom, ok := rooms[roomName]; ok {
		s.mu.RUnlock()
		return fetchedRoom
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if fetchedRoom, ok := rooms[roomName]; ok {
		return fetchedRoom
	}

	createdRoom := &room.Room{
		Peers: make(map[string]*peer.Peer),
	}
	rooms[roomName] = createdRoom

	return createdRoom
}

// Used by SFUServer to serve signals sent to it via gRPC
func (s *SFUServer) Signal(stream sfupb.SFUService_SignalServer) error {
	fmt.Println("New Signal stream connected")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client closed stream
			return nil
		}

		if err != nil {
			log.Printf("Error receiving from stream: %v", err)
			return err
		}

		roomName := req.RoomName
		userID := req.UserId
		roomByRoomName := s.getOrCreateRoom(roomName)

		if offer := req.GetOffer(); offer != nil {
			fmt.Printf("Received offer from user %s in room %s\n", userID, roomName)

			// Create PeerConnection
			peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				log.Printf("Failed to create peer connection: %v", err)
				return err
			}

			// Store the peer in the room
			roomByRoomName.Peers[userID] = &peer.Peer{
				UserID: userID,
				PeerConnection: peerConnection,
				Stream: stream,
			}

			// Set remote description (offer)
			offerSDP := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP: offer.Sdp,
			}
			if err := peerConnection.SetRemoteDescription(offerSDP); err != nil {
				log.Printf("Failed to set remote description: %v", err)
				return err
			}

			// Create answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				log.Printf("Failed to create answer: %v", err)
				return err
			}

			// Set local description
			if err := peerConnection.SetLocalDescription(answer); err != nil {
				log.Printf("Failed to set local description: %v", err)
				return err
			}

			// Send answer back
			resp := &sfupb.SignalResponse{
				RoomName: roomName,
				UserId: userID,
				Payload: &sfupb.SignalResponse_Answer{
					Answer: &sfupb.WebRTCAnswer{
						Sdp: answer.SDP,
					},
				},
			}
			if err := stream.Send(resp); err != nil {
				log.Printf("Failed to send answer: %v", err)
				return err
			}

			fmt.Printf("Sent answer to user: %s\n", userID)
		}
	}
}

func NewSFUServer() *SFUServer {
	return &SFUServer{
		Rooms: make(map[string]*room.Room),
	}
}
