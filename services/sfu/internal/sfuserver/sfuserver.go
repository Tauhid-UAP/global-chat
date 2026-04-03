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

func forwardRTP(remoteTrack *webrtc.TrackRemote, localTrack *webrtc.TrackLocalStaticRTP) {
	buf := make([]byte, 1500)

	for {
		n, _, err := remoteTrack.Read(buf)
		if err != nil {
			log.Printf("Error reading from remote track: %v\n", err)
			return
		}

		if _, err := localTrack.Write(buf[:n]); err != nil {
			log.Printf("Error writing remote data to track: %v\n", err)
			return
		}
	}
}

// SFUServer implements the gRPC SFUService
type SFUServer struct {
	sfupb.UnimplementedSFUServiceServer
	
	Rooms map[string]*room.Room
	mu sync.RWMutex

	WebRTCAPI *webrtc.API

	MaxPeersPerRoom int
}

func (s *SFUServer) GetMaxPeersPerRoom() int {
	return s.MaxPeersPerRoom
}

func (s *SFUServer) getOrCreateRoom(roomName string, maxPeers int, webRTCAPI *webrtc.API) *room.Room {
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

	createdRoom := room.CreateRoom(roomName, maxPeers, webRTCAPI)
	rooms[roomName] = createdRoom
	
	return createdRoom
}

func (s *SFUServer) deleteRoom(roomName string) {
        delete(s.Rooms, roomName)
}

func (s *SFUServer) deleteRoomIfEmpty(r *room.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.GetTotalPeers() != 0 {
		return
	}
	
	roomName := r.Name
	log.Printf("Deleting room - %s.\n", roomName)
	s.deleteRoom(roomName)
}

func (s *SFUServer) removePeerFromRoom(userID string, r *room.Room) {
	isPeerRemovedNow := r.PerformPeerRemovalOperations(userID)
	if !isPeerRemovedNow {
		log.Printf("Peer - %s is already removed.\n", userID)
		return
	}
	
	log.Printf("Peer - %s removed.\n", userID)
	s.deleteRoomIfEmpty(r)
}

// Used by SFUServer to serve signals sent to it via gRPC
func (s *SFUServer) Signal(stream sfupb.SFUService_SignalServer) error {
	log.Println("New Signal stream connected")
	
	var currentPeer *peer.Peer
	var currentRoom *room.Room
	var roomName string
	var userID string

	var hasSetRemoteDescription bool
	var bufferedICECandidates []webrtc.ICECandidateInit

	defer func() {
		if (currentPeer != nil) && (currentRoom != nil) {
			log.Printf("Cleaning up peer %s", userID)

			currentPeer.Close()
			s.removePeerFromRoom(userID, currentRoom)
		}

	}()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client closed stream
			log.Println("Stream closed by client")
			return nil
		}

		if err != nil {
			log.Printf("Error receiving from stream: %v", err)
			return err
		}
		
		if offer := req.GetOffer(); offer != nil {
			roomName = req.RoomName
			currentRoom = s.getOrCreateRoom(roomName, s.GetMaxPeersPerRoom(), s.WebRTCAPI)
			log.Println("currentRoom: ", currentRoom)
			log.Println("currentPeers: ", currentRoom.GetPeers())

			userID = req.UserId
			// log.Printf("Received offer from user %s in room %s\n", userID, roomName)
			
			newPeer, err := currentRoom.InitiatePeerForRoom(userID, stream)
			if err != nil {
				log.Printf("Error initializing peer %s: %v", userID, err)
				return err
			}
			
			// Assigned separately instead of directly to avoid redeclaration of the currentPeer variable locally which would result in nil pointer dereference later
			currentPeer = newPeer
			
			peerConnection := currentPeer.PeerConnection
			
			// ICE Trickling: Send candidates to signalling server
			peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
				// log.Println("Gathered new ICE Candidate: ", c)
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


			// Peer state monitoring
			peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
				log.Printf("Peer %s state: %s", userID, state.String())
				if state == webrtc.PeerConnectionStateConnected {
					currentRoom.SendExistingForwardedTracksToPeer(currentPeer)
					return
				}

				if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateClosed {
					currentPeer.Close()
					s.removePeerFromRoom(userID, currentRoom)
				}
			})


			// Track forwarding
			peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				log.Printf("Track received from %s", userID)

				// Make a server local track from remote peer's track.
				localTrack, err := webrtc.NewTrackLocalStaticRTP(
					remoteTrack.Codec().RTPCodecCapability,
					remoteTrack.ID(),
					fmt.Sprintf("%s-participant-%s", remoteTrack.StreamID(), userID),
				)
				if err != nil {
					log.Printf("Error creating local track for remote peer: %s | Error: %v\n", userID, err)
					return
				}

				kind := remoteTrack.Kind()
				forwardedTrack := &room.ForwardedTrack{
					Publisher: currentPeer,
					Kind: kind,
					LocalTrack: localTrack,
					RemoteTrack: remoteTrack,
				}

				currentRoom.PerformNewForwardedTrackOperations(forwardedTrack)
				
				if kind == webrtc.RTPCodecTypeVideo {
					// peer.IndicatePictureLoss(peerConnection, remoteTrack)
					peer.RequestKeyFrame(peerConnection, remoteTrack)
				}

				go forwardRTP(remoteTrack, localTrack)
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

			hasSetRemoteDescription = true
			var peerTransceivers []*peer.PeerTransceiver
			for _, transceiver := range peerConnection.GetTransceivers() {
				if transceiver.Direction() != webrtc.RTPTransceiverDirectionSendonly {
					continue
				}

				peerTransceiver := &peer.PeerTransceiver{Transceiver: transceiver}
				peerTransceivers = append(peerTransceivers, peerTransceiver)
		
				// Create a dummy local track matching the transceiver's codec kind.
				// This initializes the RTPSender's internal state (SSRC, etc.)
				var track *webrtc.TrackLocalStaticRTP
				var err error
				
				transceiverKind := transceiver.Kind()
				mid := transceiver.Mid()
				switch transceiverKind {
				case webrtc.RTPCodecTypeAudio:
					track, _ = webrtc.NewTrackLocalStaticRTP(
						webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
						fmt.Sprintf("placeholder-audio-%s", mid),
						fmt.Sprintf("placeholder-stream-%s", mid),
					)
					
				case webrtc.RTPCodecTypeVideo:
					track, _ = webrtc.NewTrackLocalStaticRTP(
						webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
						fmt.Sprintf("placeholder-video-%s", mid),
						fmt.Sprintf("placeholder-stream-%s", mid),
					)
				}

				_, err = peerConnection.AddTrack(track)
				if err != nil {
					log.Printf("Error adding dummy track: %v", err)
					return err
				}
			}

			currentPeer.PeerTransceivers = peerTransceivers

			// flush buffered ICE
			for _, candidate := range bufferedICECandidates {
				if err := peerConnection.AddICECandidate(candidate); err != nil {
					log.Printf("Error adding buffered ICE: %s | %v\n", candidate, err)
				}
			}

			bufferedICECandidates = nil

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
				log.Printf("Failed to send answer: %v\n", err)
				return err
			}

			log.Printf("Sent answer to user: %s\n", userID)

			continue
		}

		if ice := req.GetIceCandidate(); ice != nil {
			log.Println("Received remote ICE Candidate: ", ice)
			// Manually convert SdpMlineIndex to uint16 because Pion uses uint16 but Protobuf does not support uint16.
			mLineIndex := uint16(ice.SdpMlineIndex)
			
			webRTCICECandidate := webrtc.ICECandidateInit{
				Candidate: ice.Candidate,
				SDPMid: &ice.SdpMid,
				SDPMLineIndex: &mLineIndex,
			}

			if !hasSetRemoteDescription {
				bufferedICECandidates = append(bufferedICECandidates, webRTCICECandidate)
				log.Println("Buffered ICE Candidate")
				continue
			}

			err := currentPeer.PeerConnection.AddICECandidate(webRTCICECandidate)
			if err != nil {
				log.Printf("Error adding ICE candidate: %s | %v", webRTCICECandidate, err)
			}
			log.Println("Added ICE candidate from remote")
		}
	}
}

func NewSFUServer(webRTCAPI *webrtc.API, maxPeersPerRoom int) *SFUServer {
	return &SFUServer{
		Rooms: make(map[string]*room.Room),
		WebRTCAPI: webRTCAPI,
		MaxPeersPerRoom: maxPeersPerRoom,
	}
}
