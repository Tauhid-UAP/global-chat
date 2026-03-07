package room

import (
	"fmt"
	"log"
	"sync"

	"errors"

	"github.com/pion/webrtc/v3"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"

	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/peer"
)

type ForwardedTrack struct {
	PublisherID string
	Kind webrtc.RTPCodecType

	LocalTrack *webrtc.TrackLocalStaticRTP
}

// Room stores all peers in a room
type Room struct {
	Name string
	Peers map[string]*peer.Peer
	mu sync.RWMutex

	ForwardedTracks []*ForwardedTrack

	WebRTCAPI *webrtc.API

	TotalPeers int

	MaxPeers int
}

func (r *Room) GetTotalPeers() int {
	return r.TotalPeers
}

func (r *Room) GetMaxPeers() int {
	return r.MaxPeers
}

func (r *Room) IsPeerCapacityReached() bool {
	return r.GetTotalPeers() == r.GetMaxPeers()
}

func (r *Room) SetTotalPeers(totalPeers int) {
	r.TotalPeers = totalPeers
}

func (r *Room) IncrementTotalPeers(incrementBy int) {
	r.SetTotalPeers(r.GetTotalPeers() + incrementBy)
}

func (r *Room) DecrementTotalPeers(decrementBy int) {
	r.SetTotalPeers(r.GetTotalPeers() - decrementBy)
}

func (r *Room) AddPeer(p *peer.Peer) {
	r.Peers[p.UserID] = p
	r.IncrementTotalPeers(1)
}

func (r *Room) RemovePeer(userID string) {
	delete(r.Peers, userID)
}

func (r *Room) RemovePeerIfExists(userID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Peers[userID]; !ok {
		return false
	}
	
	r.RemovePeer(userID)
	r.DecrementTotalPeers(1)

	return true
}

func (r *Room) GetPeers() []*peer.Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	peers := make([]*peer.Peer, 0, len(r.Peers))
	for _, p := range r.Peers {
		peers = append(peers, p)
	}
	
	return peers
}

func (r *Room) InitiatePeerForRoom(userID string, stream sfupb.SFUService_SignalServer) (*peer.Peer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.IsPeerCapacityReached() {
		log.Println("Peer capacity reached.")
		return nil, errors.New(fmt.Sprintf("Capacity reached for room %s", r.Name))
	}
	
	// Create PeerConnection
	peerConnection, err := r.WebRTCAPI.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Printf("Failed to create peer connection: %v", err)
		return nil, err
	}

	// receivers (browser -> SFU)

	_, err = peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		return nil, err
	}

	_, err = peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		return nil, err
	}
	
	// senders (SFU -> peers)

	audioSenderSlots := []*peer.SenderSlot{}
	videoSenderSlots := []*peer.SenderSlot{}

	for i := 0; i < r.GetMaxPeers() - 1; i++ {
		audioTransceiver, err := peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeAudio,
			webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendrecv,
			},
		)
		if err != nil {
			log.Printf("Error adding audio transceiver for peer %s | %v", userID, err)
			return nil, err
		}
		audioSenderSlot := &peer.SenderSlot{
			Sender: audioTransceiver.Sender(),
			Used: false,
		}
		audioSenderSlots = append(audioSenderSlots, audioSenderSlot)

		videoTransceiver, err := peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeVideo,
			webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendrecv,
			},
		)
		if err != nil {
			log.Printf("Error adding video transceiver for peer %s | %v", userID, err)
			return nil, err
		}
		videoSenderSlot := &peer.SenderSlot{
			Sender: videoTransceiver.Sender(),
			Used: false,
		}
		videoSenderSlots = append(videoSenderSlots, videoSenderSlot)
	}
	
	newPeer := &peer.Peer{
		UserID: userID,
		PeerConnection: peerConnection,
		Stream: stream,
		AudioSenderSlots: audioSenderSlots,
		VideoSenderSlots: videoSenderSlots,
	}
	
	r.AddPeer(newPeer)

	r.SendExistingForwardedTracksToPeer(newPeer)

	return newPeer, nil
}

func (r *Room) AddForwardedTrack(forwardedTrack *ForwardedTrack) {
	r.ForwardedTracks = append(r.ForwardedTracks, forwardedTrack)
	log.Println("Added forwarded track")
}

func (r *Room) AttachForwardedTrackToPeers(forwardedTrack *ForwardedTrack) {
	log.Println("Attaching forwarded track to peers")
	for _, p := range r.Peers {
		if p.UserID == forwardedTrack.PublisherID {
			continue
		}
		
		sendForwardedTrackToPeer(forwardedTrack, p)
	}
}

func (r *Room) PerformNewForwardedTrackOperations(forwardedTrack *ForwardedTrack) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AddForwardedTrack(forwardedTrack)
	r.AttachForwardedTrackToPeers(forwardedTrack)
}

func (r *Room) SendExistingForwardedTracksToPeer(p *peer.Peer) {
	for _, forwardedTrack := range r.ForwardedTracks {
		if forwardedTrack.PublisherID == p.UserID {
			continue
		}

		sendForwardedTrackToPeer(forwardedTrack, p)
	}
}

func sendForwardedTrackToPeer(forwardedTrack *ForwardedTrack, p *peer.Peer) {
	var senderSlot *peer.SenderSlot

	switch forwardedTrack.Kind {
		case webrtc.RTPCodecTypeAudio:
			senderSlot = getFreeSenderSlot(p.AudioSenderSlots)

		case webrtc.RTPCodecTypeVideo:
			senderSlot = getFreeSenderSlot(p.VideoSenderSlots)

		default:
			return
	}

	if senderSlot == nil {
		log.Println("No sender slot")
		return
	}

	sender := senderSlot.Sender

	err := sender.ReplaceTrack(forwardedTrack.LocalTrack)
	if err != nil {
		log.Printf("Error replace track: %v\n", err)
		return
	}

	senderSlot.Used = true

	startRTCPReader(sender)
}

func getFreeSenderSlot(senderSlots []*peer.SenderSlot) *peer.SenderSlot {
	for _, ss := range senderSlots {
		if !ss.Used {
			return ss
		}
	}

	return nil
}


func startRTCPReader(sender *webrtc.RTPSender) {
	// Define and launch separate Goroutine to prevent overwriting the sender every time it is updated in the calling function
	// Otherwise, the sender variable would always be overwritten by the last value
	go func(s *webrtc.RTPSender) {
		buf := make([]byte, 1500)

		for {
			if _, _, err := s.Read(buf); err != nil {
				return
			}
		}
	}(sender)
}

func CreateRoom(roomName string, maxPeers int, webRTCAPI *webrtc.API) *Room {
	return &Room {
		Name: roomName,
		Peers: make(map[string]*peer.Peer),
		MaxPeers: maxPeers,
		WebRTCAPI: webRTCAPI,
	}
}

