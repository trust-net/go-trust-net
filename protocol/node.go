package protocol

import (
	"time"
	"sync"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/ethereum/go-ethereum/p2p"
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
	// channel for GetBlockHashesRequest
	GetBlockHashesChan chan uint8
	// channel for GetBlocksRequest
	GetBlocksChan chan uint8
	// last hash from the node
	LastHash *core.Byte64
}

// create a new instance of a node based on new peer connection
func NewNode(peer *p2p.Peer, conn p2p.MsgReadWriter) *Node {
	node := Node{
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

func (node *Node) ReadMsg() (p2p.Msg, error) {
	return node.conn.ReadMsg()
}

func (node *Node) Send(msgcode uint64, data interface{}) error {
	return p2p.Send(node.conn, msgcode, data)
}

func (node *Node) Peer() *p2p.Peer {
	return node.peer
}

func (node *Node) ID() string {
	return node.peer.ID().String()
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
