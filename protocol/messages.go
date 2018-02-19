package protocol

import (
	"github.com/trust-net/go-trust-net/core"
)

// protocol messages
const (
	// first message between peers
	Handshake = 0x00	
)


// the very first message exchanged between two byoi peers upon connection establishment
type HandshakeMsg struct {
       NetworkId core.Byte16 	`json:"network_id"       gencodec:"required"`
       ShardId core.Byte16 		`json:"shard_id"       	gencodec:"required"`
       TotalWeight core.Byte8	`json:"total_weight"  	gencodec:"required"`
}
