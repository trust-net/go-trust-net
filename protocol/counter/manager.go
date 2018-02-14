package counter

import (
	"time"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/trust-net/go-trust-net/protocol"
)

// short protocol name for handshake negotiation
var ProtocolName = "countr"

// known protocol versions
const (
	poc1 = 0x01
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

var DefaultHandshake = protocol.HandshakeMsg {
	NetworkId: *protocol.BytesToByte16([]byte{1,2,3,4}),
}
