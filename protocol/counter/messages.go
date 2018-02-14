package counter

import (
	"math/big"
	"github.com/trust-net/go-trust-net/protocol"
)

// protocol messages
const (
	// request sync
	SyncRequest = protocol.Handshake + 1
	// sync response
	SyncResponse = SyncRequest + 1
	// a new countr block is available
	NewCountrBlock = SyncResponse + 1
	// always keep last, to keep track of message count
	msgCount = NewCountrBlock + 1
)

// number of implemented messages for each supported version of the protocol
var ProtocolMsgCount = uint64(msgCount)

// a sync request message
type SyncRequestMsg struct {
	// starting hash for the sync
	StartHash []byte
	// maximum number of blocks we can handle in response
	MaxBlocks *big.Int
}

// a sync response message
type SyncResponseMsg struct {
	// an ordered of blocks
	Blocks []NewCountrBlockMsg
}


// Header implementation
type Header struct {
	Value string
}

func (h *Header) String() string {
	return h.Value
}

// NodeInfo implementation
type NodeInfo struct {
	ID string
}

func (n *NodeInfo) Id() string {
	return n.ID
}

// a new countr block message
type NewCountrBlockMsg struct {
	Previous Header
	Miner NodeInfo
	Nonce Header
	Genesis Header
	Delta  *big.Int
}
