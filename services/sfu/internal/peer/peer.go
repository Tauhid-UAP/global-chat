package peer

import (
	"log"
	"sync"
	"encoding/json"

	"github.com/pion/webrtc/v3"
	"github.com/pion/rtcp"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

type DataChannelMessage struct {
	Type string `json:"Type"`
	Data json.RawMessage `json:"Data"`
}

type PeerExitInfo struct {
	ParticipantID string `json:"ParticipantID"`
}

type TrackInfo struct {
	Mid string `json:"Mid"`
	ParticipantID string `json:"ParticipantID"`
	Kind string `json:"Kind"`
}

type PeerTransceiver struct {
	Transceiver *webrtc.RTPTransceiver
	IsUsed bool
}

// Peer represents a participant in a room
type Peer struct {
	UserID string
	PeerConnection *webrtc.PeerConnection
	DataChannel *webrtc.DataChannel
	Stream sfupb.SFUService_SignalServer
	
	PendingTrackInfo []*TrackInfo

	PeerTransceivers []*PeerTransceiver

	mu sync.Mutex
	Closed bool
}

func IndicatePictureLoss(peerConnection *webrtc.PeerConnection, track *webrtc.TrackRemote) {
	peerConnection.WriteRTCP([]rtcp.Packet{
		&rtcp.PictureLossIndication{
			MediaSSRC: uint32(track.SSRC()),
		},
	})
}

func RequestKeyFrame(peerConnection *webrtc.PeerConnection, track *webrtc.TrackRemote) {
	peerConnection.WriteRTCP([]rtcp.Packet{
		&rtcp.FullIntraRequest{
			MediaSSRC: uint32(track.SSRC()),
		},
	})
}

func SendMessageToDataChannel(dataChannelMessage *DataChannelMessage, dataChannel *webrtc.DataChannel) error {
	data, err := json.Marshal(dataChannelMessage)
	if err != nil {
		return err
	}
	
	if err := dataChannel.SendText(string(data)); err != nil {
		return err
	}

	return nil
}

func MakeDataChannelMessage(messageType string, data any) (*DataChannelMessage, error) {
	marshalledData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &DataChannelMessage{
		Type: messageType,
		Data: marshalledData,
	}, nil
}

func sendTrackInfoToDataChannel(trackInfo *TrackInfo, dataChannel *webrtc.DataChannel) error {
	dataChannelMessage, err := MakeDataChannelMessage("track-info", trackInfo)
	if err != nil {
		return err
	}

	return SendMessageToDataChannel(dataChannelMessage, dataChannel)
}

func isDataChannelOpen(dataChannel *webrtc.DataChannel) bool {
	return (dataChannel != nil) && (dataChannel.ReadyState() == webrtc.DataChannelStateOpen)
}

func (p *Peer) SendIncomingTrackInfo(trackInfo *TrackInfo) error {
	log.Println("Acquiring lock -> SendIncomingTrackInfo")
	p.mu.Lock()
	defer p.mu.Unlock()

	dataChannel := p.DataChannel
	userID := p.UserID
	if !isDataChannelOpen(dataChannel) {
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
	log.Println("Acquiring lock -> FlushPendingTrackInfo")
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
	log.Println("Acquiring lock -> Close")
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
