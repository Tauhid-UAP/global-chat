package peer

import (
	"github.com/pion/webrtc/v3"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

// Peer represents a participant in a room
type Peer struct {
	UserID string
	PeerConnection *webrtc.PeerConnection
	Stream sfupb.SFUService_SignalServer
}
