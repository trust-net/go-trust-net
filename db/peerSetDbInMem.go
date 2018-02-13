package db

import (
	"sync"
)


// an in memory implementation of PeerSetDB interface
type PeerSetDbInMemory struct {
	nodes map[string]PeerNode
	myHistory map[interface{}]bool
	lock   sync.RWMutex
}

// create an instance
func NewPeerSetDbInMemory() PeerSetDb {
	ps := &PeerSetDbInMemory{}
	ps.nodes = make(map[string]PeerNode)
	ps.myHistory = make(map[interface{}]bool)
	return ps
}

func (ps *PeerSetDbInMemory) PeerNodeForId(id string) PeerNode {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.nodes[id]
}

func (ps *PeerSetDbInMemory) RegisterPeerNode(node PeerNode) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.nodes[node.Id()] = node
	return nil
}

func (ps *PeerSetDbInMemory) UnRegisterPeerNodeForId(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	delete(ps.nodes, id)
	return nil
}

func (ps *PeerSetDbInMemory) PeerNodesWithMsgNotSeen(item interface{}) []PeerNode {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	peers := make([]PeerNode, 0)
	for _, node := range ps.nodes {
		if !node.HasTx(item) {
			peers = append(peers, node) 
		}
	}
	return peers
}

func (ps *PeerSetDbInMemory) HaveISeenIt(item interface{}) bool {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	seen := ps.myHistory[item]
	if !seen {
		ps.myHistory[item] = true
	}
	return seen
}
