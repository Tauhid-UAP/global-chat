package room

import (
	"sync"

	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/peer"
)

// Room stores all peers in a room
type Room struct {
	Peers map[string]*peer.Peer
	mu sync.Mutex
}
