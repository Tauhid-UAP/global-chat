package peer

import (
	"sync"

	"github.com/pion/webrtc/v3"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

// Peer represents a participant in a room
type Peer struct {
	UserID string
	PeerConnection *webrtc.PeerConnection
	Stream sfupb.SFUService_SignalServer

	mu sync.Mutex
	Closed bool
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
