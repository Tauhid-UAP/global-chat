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

	createdRoom := room.CreateRoom(roomName)
	rooms[roomName] = createdRoom
	
	return createdRoom
}

// Used by SFUServer to serve signals sent to it via gRPC
func (s *SFUServer) Signal(stream sfupb.SFUService_SignalServer) error {
	log.Println("New Signal stream connected")

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
		
		var currentPeer *peer.Peer;

		roomName := req.RoomName
		currentRoom := s.getOrCreateRoom(roomName)
		

		if offer := req.GetOffer(); offer != nil {
			userID := req.UserId
			fmt.Printf("Received offer from user %s in room %s\n", userID, roomName)

			// Create PeerConnection
			peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				log.Printf("Failed to create peer connection: %v", err)
				return err
			}
			
			currentPeer = &peer.Peer{
				UserID: userID,
				PeerConnection: peerConnection,
				Stream: stream,
			}

			currentRoom.AddPeer(currentPeer)

			// ICE Trickling: Send candidates to signalling server
			peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c == nil {
					return
				}

				deserializedCandidate := c.ToJSON()

				response := &sfupb.SignalResponse{
					RoomName: roomName,
					UserId: userID,
					Payload: &sfupb.SignalResponse_IceCandidate{
						IceCandidate: &sfupb.WebRTCICECandidate{
							Candidate: deserializedCandidate.Candidate,
							SdpMid: *deserializedCandidate.SDPMid,
							SdpMlineIndex: int32(*deserializedCandidate.SDPMLineIndex),

						},

					},
				}

				if err := stream.Send(response); err != nil {
					log.Println("ICE send error:", err)

				}
			})

			// Track forwarding
			peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				log.Printf("Track received from %s", userID)

				for _, storedPeer := range currentRoom.GetPeers() {
					if storedPeer.UserID == userID {
						// Ignore track sent by the current peer as they already have their own track.
						continue

					}
					
					// Make a server local track from remote peer's track.
					localTrack, err := webrtc.NewTrackLocalStaticRTP(
						remoteTrack.Codec().RTPCodecCapability,
						remoteTrack.ID(),
						remoteTrack.StreamID(),
					)
					if err != nil {
						continue
					}
					
					// Add the 'local' track to the current peer. The local track is actually a copy of the remote track.
					_, err = storedPeer.PeerConnection.AddTrack(localTrack)
					if err != nil {
						continue
					}

					go func() {
						buf := make([]byte, 1500)
						for {
							// Keep reading from the remote track.
							n, _, readErr := remoteTrack.Read(buf)
							if readErr != nil {
								return
							}
							
							// Whatever is read from the remote track is written to the local copy.
							if _, writeErr := localTrack.Write(buf[:n]); writeErr != nil {
								return
							}
						}
					}()

				}
			})

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
			response := &sfupb.SignalResponse{
				RoomName: roomName,
				UserId: userID,
				Payload: &sfupb.SignalResponse_Answer{
					Answer: &sfupb.WebRTCAnswer{
						Sdp: answer.SDP,
					},
				},
			}
			if err := stream.Send(response); err != nil {
				log.Printf("Failed to send answer: %v", err)
				return err
			}

			fmt.Printf("Sent answer to user: %s\n", userID)

			continue
		}

		if ice := req.GetIceCandidate(); ice != nil {
			// Manually convert SdpMlineIndex to uint16 because Pion uses uint16 but Protobuf does not support uint16.
			mLineIndex := uint16(ice.SdpMlineIndex)
			err := currentPeer.PeerConnection.AddICECandidate(
				webrtc.ICECandidateInit{
					Candidate: ice.Candidate,
					SDPMid: &ice.SdpMid,
					SDPMLineIndex: &mLineIndex,
				},
			)
			if err != nil {
				log.Println("Error adding ICE candidate:", err)
			}
		}
	}
}

func NewSFUServer() *SFUServer {
	return &SFUServer{
		Rooms: make(map[string]*room.Room),
	}
}
