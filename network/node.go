package network

import (
	"time"
	"sync"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/ethereum/go-ethereum/p2p"
//	"github.com/ethereum/go-ethereum/p2p/discover"
)


const (
	maxKnownTxs      = 1024 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks   = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	handshakeTimeout = 5 * time.Second
)

const (
	// default value will be used for error, since closed channel sends default message
	CHAN_ERROR = uint8(0)
	CHAN_NEXT = uint8(1)
	CHAN_ABORT = uint8(2)
	CHAN_RETRY = uint8(3)
	CHAN_DONE = uint8(4)
)

// extracting peer node interface, so can implement mock for testing
// (see mockNode in test_fixtures.go)
type PeerNode interface {
	ReadMsg() (p2p.Msg, error)
	Send(msgcode uint64, data interface{}) error
	Peer() *p2p.Peer
	Conn() p2p.MsgReadWriter
	Status() *AppConfig
	NodeId() core.Byte64
	Id() string
	Name() string
	SetStatus(status *AppConfig)
	AddTx(tx interface{})
	HasTx(tx interface{}) bool
	AddBlock(block interface{})
	HasBlock(block interface{}) bool
}

type peerNode struct {
	// reference to p2p peer info
	peer	 *p2p.Peer
	// reference to read/write connection
	conn	 p2p.MsgReadWriter
	// reference to status of the node
	status *AppConfig
	// Set of transactions known to be known by this peer
	knownTxs    *common.Set
	// Set of blocks known to be known by this peer 
	knownBlocks *common.Set
	// synchronization mutex
	lock sync.RWMutex	
	// channel for GetBlockHashesRequest
	GetBlockHashesChan chan uint8
	// channel for GetBlocksRequest
	GetBlocksChan chan uint8
	// last hash from the node
	LastHash *core.Byte64
}

// create a new instance of a node based on new peer connection
func NewPeerNode(peer *p2p.Peer, conn p2p.MsgReadWriter) *peerNode {
	node := peerNode{
		peer: peer,
		conn: conn,
		knownTxs: common.NewSet(),
		knownBlocks: common.NewSet(),
		// we are using buffered channel, to unblock sync go routines when peer disconnects while we are stil syncing
		GetBlockHashesChan: make(chan uint8, 2),
		GetBlocksChan: make(chan uint8, 2),
	}
	return &node
}

func (node *peerNode) ReadMsg() (p2p.Msg, error) {
	return node.conn.ReadMsg()
}

func (node *peerNode) Send(msgcode uint64, data interface{}) error {
	return p2p.Send(node.conn, msgcode, data)
}

func (node *peerNode) Peer() *p2p.Peer {
	return node.peer
}

func (node *peerNode) Conn() p2p.MsgReadWriter {
	return node.conn
}

func (node *peerNode) Status() *AppConfig {
	return node.status
}

func (node *peerNode) NodeId() core.Byte64 {
	return *core.BytesToByte64(node.peer.ID().Bytes())
}

func (node *peerNode) Id() string {
	return node.peer.ID().String()
}

func (node *peerNode) Name() string {
	return node.peer.Name()
}

func (node *peerNode) SetStatus(status *AppConfig) {
	node.status = status
}

func (node *peerNode) AddTx(tx interface{}) {
	if node.knownTxs.Size() >= maxKnownTxs {
		node.lock.Lock()
		defer node.lock.Unlock()
		// pop a few older transactions
		node.knownTxs.Pop()
		node.knownTxs.Pop()
		node.knownTxs.Pop()
	}
	node.knownTxs.Add(tx)
}

func (node *peerNode) HasTx(tx interface{}) bool{
	return node.knownTxs.Has(tx)
}

func (node *peerNode) AddBlock(block interface{}) {
	if node.knownBlocks.Size() >= maxKnownBlocks {
		node.lock.Lock()
		defer node.lock.Unlock()
		// pop a few older transactions
		node.knownBlocks.Pop()
		node.knownBlocks.Pop()
		node.knownBlocks.Pop()
	}
	node.knownBlocks.Add(block)
}

func (node *peerNode) HasBlock(block interface{}) bool{
	return node.knownBlocks.Has(block)
}
