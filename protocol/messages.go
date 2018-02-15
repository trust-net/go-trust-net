package protocol

import (
	"math/big"
	"github.com/trust-net/go-trust-net/core"
)

// protocol messages
const (
	// first message between peers
	Handshake = 0x00	
)


// the very first message exchanged between two byoi peers upon connection establishment
type HandshakeMsg struct {
	NetworkId	core.Byte16
	ShardId		core.Byte16
	TD			*big.Int
	CurrentBlock    core.Byte32
	GenesisBlock    core.Byte32
}
