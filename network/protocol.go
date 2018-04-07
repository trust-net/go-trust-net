package network

import (
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/consensus"
)

// protocol messages
const (
	// first message between peers
	Handshake = 0x00	
	// request to get block hashes during handshake
	GetBlockHashesRequest = Handshake + 1
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

// protocol errors
const (
	ErrorHandshakeFailed = 0x01
	ErrorMaxPeersReached = 0x02
	ErrorUnknownMessageType = 0x03
	ErrorNotImplemented = 0x04
	ErrorSyncFailed = 0x05
	ErrorInvalidRequest = 0x06
	ErrorInvalidResponse = 0x07
	ErrorNotFound = 0x08
	ErrorBadBlock = 0x09
)

// number of implemented messages for each supported version of the protocol
var ProtocolMsgCount = uint64(msgCount)

// the very first message exchanged between two byoi peers upon connection establishment
type HandshakeMsg struct {
	// following 3 fields need exact match
    NetworkId core.Byte16 	`json:"network_id"       gencodec:"required"`
//    ShardId core.Byte16 		`json:"shard_id"       	gencodec:"required"`
    Genesis core.Byte64		`json:"genesis"  		gencodec:"required"`
    // application specific protocol compatibility match
	ProtocolId core.Byte16	`json:"prorocol_id"  	gencodec:"required"`
//    // application specific authentication/authorization
//	AuthToken core.Byte64	`json:"auth_token"	  	gencodec:"required"`
    // following 2 fields are used for fork resolution
    TotalWeight core.Byte8	`json:"total_weight"  	gencodec:"required"`
    TipNumeric core.Byte8	`json:"tip_numeric"  	gencodec:"required"`
}

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
type GetBlocksResponseMsg []consensus.BlockSpec

// Announce a new countr block
//type NewBlockMsg core.BlockSpec
type NewBlockMsg consensus.BlockSpec
