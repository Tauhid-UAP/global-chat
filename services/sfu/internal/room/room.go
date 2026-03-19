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
	Publisher *peer.Peer
	Kind webrtc.RTPCodecType

	LocalTrack *webrtc.TrackLocalStaticRTP
	RemoteTrack *webrtc.TrackRemote
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
	if _, ok := r.Peers[userID]; !ok {
		return false
	}
	
	r.RemovePeer(userID)
	r.DecrementTotalPeers(1)

	return true
}

func (r *Room) PerformPeerRemovalOperations(userID string) bool {
	log.Println("Acquiring lock -> PerformPeerRemovalOperations")
	r.mu.Lock()
	defer r.mu.Unlock()

	isRemovedNow := r.RemovePeerIfExists(userID)
	if !isRemovedNow {
		return false
	}

	peerExitInfo := &peer.PeerExitInfo{
		ParticipantID: userID,
	}
	dataChannelMessage, err := peer.MakeDataChannelMessage("peer-exit-info", peerExitInfo)
	if err != nil {
		log.Println("Error making data channel message with peer exit info: %v", err)
		return true
	}

	for _, p := range r.Peers {
		err = peer.SendMessageToDataChannel(dataChannelMessage, p.DataChannel)
		if err == nil {
			continue
		}

		log.Println("Error sending peer exit info to data channel: %v", err)
	}

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
	log.Println("Acquiring lock -> InitiatePeerForRoom")
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

	newPeer := &peer.Peer{
		UserID: userID,
		PeerConnection: peerConnection,
		Stream: stream,
	}

	peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		log.Printf("Data channel received for peer - %s\n", userID)
		newPeer.DataChannel = dataChannel
	
		dataChannel.OnOpen(func() {
			log.Printf("Data channel opened for peer - %s\n", userID)
			// Store the channel on the peer for later use
	
			total, failureCount, errs := newPeer.FlushPendingTrackInfo()
			log.Printf("Flushed %d track info for peer - %s\n", total, userID)
			if failureCount > 0 {
				for _, err := range errs {
					log.Printf("Flush error: %v\n", err)
				}
			}
		})
	})
	
	r.AddPeer(newPeer)

	return newPeer, nil
}

func (r *Room) AddForwardedTrack(forwardedTrack *ForwardedTrack) {
	r.ForwardedTracks = append(r.ForwardedTracks, forwardedTrack)
	log.Println("Added forwarded track")
}

func (r *Room) SendForwardedTrackToPeers(forwardedTrack *ForwardedTrack) {
	log.Println("Sending forwarded track to all peers")
	for _, p := range r.Peers {
		userID := p.UserID
		if userID == forwardedTrack.Publisher.UserID {
			continue
		}
		
		if err := sendForwardedTrackToPeer(forwardedTrack, p); err != nil {
			log.Printf("Error sending forwarded track to peer - %s | %v\n", userID, err)
			continue
		}

		log.Printf("Sent forwarded track to peer - %s\n", userID)
	}
}

func (r *Room) PerformNewForwardedTrackOperations(forwardedTrack *ForwardedTrack) {
	log.Println("Acquiring lock -> PerformNewForwardedTrackOperations")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AddForwardedTrack(forwardedTrack)
	r.SendForwardedTrackToPeers(forwardedTrack)
}

func sendForwardedTrackToPeer(forwardedTrack *ForwardedTrack, p *peer.Peer) error {
	// Used for sending exisitng (already forwarded) tracks to new peer

	peerConnection := p.PeerConnection

	for _, transceiver := range peerConnection.GetTransceivers() {
		log.Printf(
			"kind=%s direction=%s mid=%s sender=%v",
			transceiver.Kind(), transceiver.Direction(), transceiver.Mid(), transceiver.Sender())

		if transceiver.Direction() != webrtc.RTPTransceiverDirectionSendonly {
			continue
		}

		forwardedTrackKind := forwardedTrack.Kind
		log.Println("Required kind: ", forwardedTrackKind)
		if transceiver.Kind() != forwardedTrackKind {
			continue
		}

		sender := transceiver.Sender()

		if err := sender.ReplaceTrack(forwardedTrack.LocalTrack); err != nil {
			return err
		}

		startRTCPReader(sender)

		mid := transceiver.Mid()
		trackInfo := &peer.TrackInfo{
			Mid: mid,
			ParticipantID: forwardedTrack.Publisher.UserID,
			Kind: forwardedTrackKind.String(),
		}

		p.SendIncomingTrackInfo(trackInfo)

		return nil
	}

	return errors.New("No transceiver exists for sending")
}

func (r *Room) SendExistingForwardedTracksToPeer(p *peer.Peer) {
	log.Println("Acquiring lock -> SendExistingForwardedTracksToPeer")
	r.mu.Lock()
	defer r.mu.Unlock()
	userID := p.UserID
	for _, forwardedTrack := range r.ForwardedTracks {
		publisher := forwardedTrack.Publisher
		if publisher.UserID == userID {
			continue
		}

		if err := sendForwardedTrackToPeer(forwardedTrack, p); err != nil {
			log.Printf("Error sending existing forwarded track to peer - %s | %v\n", userID, err)
			continue
		}

		log.Printf("Sent existing forwarded track to peer - %s\n", userID)

		if forwardedTrack.Kind != webrtc.RTPCodecTypeVideo {
			continue
		}

		peer.IndicatePictureLoss(publisher.PeerConnection, forwardedTrack.RemoteTrack)

		log.Println("Indicated picture loss")
	}
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

