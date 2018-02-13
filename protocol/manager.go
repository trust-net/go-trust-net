package protocol

import (
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type ProtocolManager interface {
	// provide an instance of p2p protocol implementation
	Protocol() p2p.Protocol
	
	// initiate connection and handshake with a node
	AddPeer(node *discover.Node) error

	// perform protocol specific handshake with newly connected peer
	Handshake(peer *p2p.Peer, ws p2p.MsgReadWriter) error
	
	// get reference to protocol manager's DB
	Db() db.PeerSetDb
}

// protocol errors
const (
	ErrorHandshakeFailed = 0x01
	ErrorMaxPeersReached = 0x02
	ErrorUnknownMessageType = 0x03
	ErrorNotImplemented = 0x04
)
