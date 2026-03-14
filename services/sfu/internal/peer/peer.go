package peer

import (
	"log"
	"sync"
	"encoding/json"

	"github.com/pion/webrtc/v3"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

type SenderSlot struct {
        Sender *webrtc.RTPSender
		Mid string
        Used bool
}

type TrackInfo struct {
	Type string `json:"Type"`
	Mid string `json:"Mid"`
	ParticipantID string `json:"ParticipantID"`
	Kind string `json:"Kind"`
}


// Peer represents a participant in a room
type Peer struct {
	UserID string
	PeerConnection *webrtc.PeerConnection
	DataChannel *webrtc.DataChannel
	Stream sfupb.SFUService_SignalServer
	
	PendingTrackInfo []*TrackInfo
	AudioSenderSlots []*SenderSlot
	VideoSenderSlots []*SenderSlot

	mu sync.Mutex
	Closed bool
}

func sendTrackInfoToDataChannel(trackInfo *TrackInfo, dataChannel *webrtc.DataChannel) error {
	data, err := json.Marshal(trackInfo)
	if err != nil {
		return err
	}
	
	if err := dataChannel.SendText(string(data)); err != nil {
		return err
	}

	return nil
}

func (p *Peer) SendIncomingTrackInfo(trackInfo *TrackInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dataChannel := p.DataChannel
	userID := p.UserID
	if (dataChannel == nil) || (dataChannel.ReadyState() != webrtc.DataChannelStateOpen) {
		p.PendingTrackInfo = append(p.PendingTrackInfo, trackInfo)
		log.Printf("Data channel not open for peer - %s | Track queued\n", userID)
		return nil
	}

	if err := sendTrackInfoToDataChannel(trackInfo, dataChannel); err != nil {
		return err
	}

	log.Printf("Sent track to peer - %s\n", userID)
	return nil
}

func (p *Peer) FlushPendingTrackInfo() (int, int, []error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	dataChannel := p.DataChannel
	total := 0
	failed := 0
	var errs []error
	for _, trackInfo := range p.PendingTrackInfo {
		total++
		if err := sendTrackInfoToDataChannel(trackInfo, dataChannel); err != nil {
			failed++
			errs = append(errs, err)
			continue
		}
	}

	p.PendingTrackInfo = nil

	return total, failed, errs
}

func (p *Peer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Closed {
		return
	}

	p.Closed = true
	
	peerConnection := p.PeerConnection
	if peerConnection == nil {
		return
	}

	peerConnection.Close()
}
