package room

import (
	"sync"

	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/peer"
)

// Room stores all peers in a room
type Room struct {
	Name string
	Peers map[string]*peer.Peer
	mu sync.RWMutex

	TotalPeers int
}

func (r *Room) GetTotalPeers() int {
	return r.TotalPeers
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
	r.mu.Lock()
	defer r.mu.Unlock()
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

func CreateRoom(roomName string) *Room {
	return &Room {
		Name: roomName,
		Peers: make(map[string]*peer.Peer),
	}
}
