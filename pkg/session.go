package sfu

import (
	"sync"

	log "github.com/pion/ion-log"
)

// Session represents a set of peers. Transports inside a session
// are automatically subscribed to each other.
type Session struct {
	id             string
	mu             sync.RWMutex
	peers          map[string]*Peer
	onCloseHandler func()
	closed         bool
}

// NewSession creates a new session
func NewSession(id string) *Session {
	return &Session{
		id:     id,
		peers:  make(map[string]*Peer),
		closed: false,
	}
}

// AddPublisher adds a transport to the session
func (s *Session) AddPeer(peer *Peer) {
	s.mu.Lock()
	s.peers[peer.id] = peer
	s.mu.Unlock()
}

// RemovePeer removes a transport from the session
func (s *Session) RemovePeer(pid string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Infof("RemovePeer %s from session %s", pid, s.id)
	delete(s.peers, pid)

	// Close session if no peers
	if len(s.peers) == 0 && s.onCloseHandler != nil && !s.closed {
		s.onCloseHandler()
		s.closed = true
	}
}

// Publish will add a Sender to all peers in current Session from given
// Receiver
func (s *Session) Publish(router Router, rr *receiverRouter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for pid, p := range s.peers {
		// Don't sub to self
		if router.ID() == pid {
			continue
		}

		log.Infof("Publish mediaSSRC to %s", pid)

		if err := router.AddSender(p.subscriber, rr); err != nil {
			log.Errorf("Error subscribing transport to router: %s", err)
			continue
		}
	}
}

// Subscribe will create a Sender for every other Receiver in the session
func (s *Session) Subscribe(peer *Peer) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for pid, p := range s.peers {
		if pid == peer.id {
			continue
		}
		err := p.publisher.GetRouter().AddSender(peer.subscriber, nil)
		if err != nil {
			log.Errorf("Subscribing to router err: %v", err)
			continue
		}
	}
}

// Transports returns peers in this session
func (s *Session) Peers() map[string]*Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peers
}

// OnClose is called when the session is closed
func (s *Session) OnClose(f func()) {
	s.onCloseHandler = f
}
