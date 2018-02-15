package pager

import (
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/protocol"
)

// protocol messages
const (
	// a broadcast message for text messages during poc
	BroadcastText = protocol.Handshake + 1
	// always keep last, to keep track of message count
	msgCount = BroadcastText + 1
)

// number of implemented messages for each supported version of the protocol
var ProtocolMsgCount = uint64(msgCount)

// a broadcast message for text message on the network during poc
type BroadcastTextMsg struct {
	MsgId core.Byte16
	MsgText string
}
