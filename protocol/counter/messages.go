package counter

import (
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/protocol"
	"github.com/trust-net/go-trust-net/consensus"
)

// protocol messages
const (
	// request to get block hashes during handshake
	GetBlockHashesRequest = protocol.Handshake + 1
	// response with block hashes
	GetBlockHashesResponse = GetBlockHashesRequest + 1
	// peer is not on main blockchain and needs to rewind to earlier block
	GetBlockHashesRewind = GetBlockHashesResponse + 1
	// request to get blocks
	GetBlocksRequest = GetBlockHashesRewind + 1
	// response to get blocks
	GetBlocksResponse = GetBlocksRequest + 1
	// new block accouncement
	NewBlock = GetBlocksResponse + 1
	// always keep last, to keep track of message count
	msgCount = NewBlock + 1
)

// opcodes for block spec
const (
	opIncrement = uint64(1)
	opDecrement = uint64(2)
)
var OpIncrement = core.Uint64ToByte8(opIncrement)
var OpDecrement = core.Uint64ToByte8(opDecrement)

// number of implemented messages for each supported version of the protocol
var ProtocolMsgCount = uint64(msgCount)

// request block hashes for descendents of a parent
type GetBlockHashesRequestMsg struct {
	// starting block's parent's hash
	ParentHash core.Byte64 	`json:"parent_hash"      gencodec:"required"`
	// maximum number of blocks we can handle in response
	MaxBlocks core.Byte8		`json:"max_blocks"       gencodec:"required"`
}

// response with array of block hashes
type GetBlockHashesResponseMsg []core.Byte64

// node needs to rewind back to specified block
type GetBlockHashesRewindMsg core.Byte64

// request block specs for specified hashes
type GetBlocksRequestMsg []core.Byte64

// response with array of requested block specs
//type GetBlocksResponseMsg []*core.BlockSpec
type GetBlocksResponseMsg []consensus.Block

// Announce a new countr block
//type NewBlockMsg core.BlockSpec
type NewBlockMsg consensus.Block

//
//// Header implementation
//type Header struct {
//	Value string
//}
//
//func (h *Header) String() string {
//	return h.Value
//}
//
//// NodeInfo implementation
//type NodeInfo struct {
//	ID string
//}
//
//func (n *NodeInfo) Id() string {
//	return n.ID
//}
//
//// a new countr block message
//type NewCountrBlockMsg struct {
//	Previous Header
//	Miner NodeInfo
//	Nonce Header
//	Genesis Header
//	Delta  *big.Int
//}
