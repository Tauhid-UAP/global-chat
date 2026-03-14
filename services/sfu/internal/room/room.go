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

	// Create data channel for sending track metadata to the browser
	// dataChannel, err := peerConnection.CreateDataChannel("track-info", nil)
	// if err != nil {
	// 	log.Printf("Failed to create data channel for peer - %s | %v\n", userID, err)
	// 	return nil, err
	// }

	// audioSenderSlots := []*peer.SenderSlot{}
	// videoSenderSlots := []*peer.SenderSlot{}
	// for i := 0; i < r.GetMaxPeers(); i++ {
	// 	audioTrack, _ := webrtc.NewTrackLocalStaticRTP(
	// 		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
	// 		"audio",
	// 		"sfu",
	// 	)

	// 	peerConnection.AddTrack(audioTrack)

	// 	// audioSender, _ := peerConnection.AddTrack(audioTrack)

	// 	// audioSender.ReplaceTrack(nil)
	
	// 	videoTrack, _ := webrtc.NewTrackLocalStaticRTP(
	// 		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
	// 		"video",
	// 		"sfu",
	// 	)

	// 	peerConnection.AddTrack(videoTrack)

	// 	// videoSender, _ := peerConnection.AddTrack(videoTrack)
	
	// 	// videoSender.ReplaceTrack(nil)
	// }

	/*
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
	*/

	newPeer := &peer.Peer{
		UserID: userID,
		PeerConnection: peerConnection,
		Stream: stream,
		// DataChannel: dataChannel,
		// AudioSenderSlots: []*peer.SenderSlot{},
		// VideoSenderSlots: []*peer.SenderSlot{},
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

	// dataChannel.OnOpen(func() {
	// 	log.Printf("Data channel opened for peer - %s\n", userID)

	// 	total, failure_count, errs := newPeer.FlushPendingTrackInfo()

	// 	log.Printf("Successfully flushed %d track info for peer - %s\n", total, userID)

	// 	if failure_count > 0 {
	// 		log.Printf("Failed to flush %d track info for peer - %s\n", failure_count, userID)
	// 		for _, err := range errs {
	// 			log.Printf("Flush error: %v\n", err)
	// 		}
	// 	}
	// })
	
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
		if userID == forwardedTrack.PublisherID {
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AddForwardedTrack(forwardedTrack)
	r.SendForwardedTrackToPeers(forwardedTrack)
}

func (r *Room) SendExistingForwardedTracksToPeer(p *peer.Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	userID := p.UserID
	for _, forwardedTrack := range r.ForwardedTracks {
		if forwardedTrack.PublisherID == userID {
			continue
		}

		if err := sendForwardedTrackToPeer(forwardedTrack, p); err != nil {
			log.Printf("Error sending existing forwarded track to peer - %s | %v\n", userID, err)
			continue
		}

		log.Printf("Sent existing forwarded track to peer - %s\n", userID)
	}
}

// func sendForwardedTrackToPeer(forwardedTrack *ForwardedTrack, p *peer.Peer) error {
// 	var senderSlot *peer.SenderSlot

// 	forwardedTrackKind := forwardedTrack.Kind
// 	switch forwardedTrackKind {
// 		case webrtc.RTPCodecTypeAudio:
// 			senderSlot = getFreeSenderSlot(p.AudioSenderSlots)

// 		case webrtc.RTPCodecTypeVideo:
// 			senderSlot = getFreeSenderSlot(p.VideoSenderSlots)

// 		default:
// 			return errors.New("Invalid kind: " + forwardedTrackKind.String())
// 	}

// 	if senderSlot == nil {
// 		return errors.New("No sender slot")
// 	}

// 	sender := senderSlot.Sender

// 	err := sender.ReplaceTrack(forwardedTrack.LocalTrack)
// 	if err != nil {
// 		return err
// 	}

// 	senderSlot.Used = true

// 	startRTCPReader(sender)

// 	mid := senderSlot.Mid
// 	trackInfo := &peer.TrackInfo{
// 		Type: "track-info",
// 		Mid: mid,
// 		ParticipantID: forwardedTrack.PublisherID,
// 		Kind: forwardedTrackKind.String(),
// 	}

// 	p.SendIncomingTrackInfo(trackInfo)

// 	return nil
// }

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

		// if transceiver.Sender() == nil {
		// 	continue
		// }
		sender := transceiver.Sender()

		// log.Println("Track: ", sender)
		// if sender.Track() != nil {
		// 	continue
		// }

		if err := sender.ReplaceTrack(forwardedTrack.LocalTrack); err != nil {
			return err
		}

		startRTCPReader(sender)

		mid := transceiver.Mid()
		trackInfo := &peer.TrackInfo{
			Type: "track-info",
			Mid: mid,
			ParticipantID: forwardedTrack.PublisherID,
			Kind: forwardedTrackKind.String(),
		}

		p.SendIncomingTrackInfo(trackInfo)

		return nil
	}

	return errors.New("No transceiver exists for sending")
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

