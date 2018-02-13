package protocol

import (
	"math/big"	
)

// short protocol name for handshake negotiation
var ProtocolName = "pager"

// known protocol versions
const (
	poc1 = 0x01
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

// protocol messages
const (
	// first message between peers
	Handshake = 0x00	
	// a broadcast message for text messages during poc
	BroadcastText = Handshake + 1
	// always keep last, to keep track of message count
	msgCount = BroadcastText + 1
)

// number of implemented messages for each supported version of the protocol
var ProtocolMsgCount = uint64(msgCount)

// the very first message exchanged between two byoi peers upon connection establishment
type HandshakeMsg struct {
	NetworkId	Byte16
	ShardId		Byte16
	TD			*big.Int
	CurrentBlock    Byte32
	GenesisBlock    Byte32
}

// a broadcast message for text message on the network during poc
type BroadcastTextMsg struct {
	MsgId Byte16
	MsgText string
}
