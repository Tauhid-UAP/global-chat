package room

import (
	"sync"

	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/peer"
)

// Room stores all peers in a room
type Room struct {
	Peers map[string]*peer.Peer
	mu sync.RWMutex
}

func (r *Room) AddPeer(p *peer.Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Peers[p.UserID] = p
}

func (r *Room) RemovePeer(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Peers, userID)
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

func CreateRoom(roomName string) *Room {
	return &Room {
		Peers: make(map[string]*peer.Peer),
	}
}
