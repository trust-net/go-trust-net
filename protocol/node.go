package protocol

import (
	"time"
	"sync"
	"github.com/trust-net/go-trust-net/common"
	"github.com/ethereum/go-ethereum/p2p"
)


const (
	maxKnownTxs      = 1024 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks   = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	handshakeTimeout = 5 * time.Second
)

type Node struct {
	// reference to p2p peer info
	peer	 *p2p.Peer
	// reference to read/write connection
	conn	 p2p.MsgReadWriter
	// reference to status of the node
	status *HandshakeMsg
	// Set of transactions known to be known by this peer
	knownTxs    *common.Set
	// Set of blocks known to be known by this peer 
	knownBlocks *common.Set
	// synchronization mutex
	lock sync.RWMutex
}

// create a new instance of a node based on new peer connection
func NewNode(peer *p2p.Peer, conn p2p.MsgReadWriter) *Node {
	node := Node{
		peer: peer,
		conn: conn,
		knownTxs: common.NewSet(),
		knownBlocks: common.NewSet(),
	}
	return &node
}

func (node *Node) Peer() *p2p.Peer {
	return node.peer
}

func (node *Node) Conn() p2p.MsgReadWriter {
	return node.conn
}

func (node *Node) Status() *HandshakeMsg {
	return node.status
}

func (node *Node) Id() string {
	return node.peer.ID().String()
}

func (node *Node) SetStatus(status *HandshakeMsg) {
	node.status = status
}

func (node *Node) AddTx(tx interface{}) {
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

func (node *Node) HasTx(tx interface{}) bool{
	return node.knownTxs.Has(tx)
}

func (node *Node) AddBlock(block interface{}) {
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

func (node *Node) HasBlock(block interface{}) bool{
	return node.knownBlocks.Has(block)
}
