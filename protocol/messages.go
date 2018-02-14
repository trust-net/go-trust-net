package protocol

import (
	"math/big"
)

// protocol messages
const (
	// first message between peers
	Handshake = 0x00	
)


// the very first message exchanged between two byoi peers upon connection establishment
type HandshakeMsg struct {
	NetworkId	Byte16
	ShardId		Byte16
	TD			*big.Int
	CurrentBlock    Byte32
	GenesisBlock    Byte32
}
