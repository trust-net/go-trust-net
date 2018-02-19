package pager

import (
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/go-trust-net/protocol"
)

// short protocol name for handshake negotiation
var ProtocolName = "pager"

const (
	// current protocol version
	poc1 = 0x01
	// maximum number of peers to connect at any given time
	maxPeers = 10
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

var defaultHandshake = protocol.HandshakeMsg {
	NetworkId: *core.BytesToByte16([]byte{1,2,3,4}),
}

// signature for callback function when a page is recieved
type PageHandler func (from, txt string)

// a "pager" protocol manager implementation
type PagerProtocolManager struct {
	protocol.ManagerBase
	logger log.Logger
	callback PageHandler
}

// create a new instance of pager protocol manager
func NewPagerProtocolManager(callback PageHandler) *PagerProtocolManager {
	mgr := PagerProtocolManager{
		callback: callback,
	}
	mgr.SetDb(db.NewPeerSetDbInMemory())
	mgr.logger = log.NewLogger(mgr)
	mgr.logger.Debug("Created new instance of pager protocol manager")
	return &mgr
}

func (mgr *PagerProtocolManager) Broadcast(msg BroadcastTextMsg) int {
	count := 0
	for _, node := range mgr.Db().PeerNodesWithMsgNotSeen(msg.MsgId) {
		peer, _ := node.(*protocol.Node)
		// first mark this peer as has seen this message, so we can stop cyclic receive immediately
		peer.AddTx(msg.MsgId)
		p2p.Send(peer.Conn(), BroadcastText, msg)
		mgr.logger.Debug("relayed message '%s' to %s", msg.MsgText, peer.Peer().Name())
		count++
	}
	return count
}

func (mgr *PagerProtocolManager) handleBroadcastMsg(msg p2p.Msg, from *p2p.Peer) error {
	var peerMessage BroadcastTextMsg
	err := msg.Decode(&peerMessage)
	if err != nil {
		// doing nothing, just skip
		mgr.logger.Error("%s", err)
	} else {
		// first check if this is not a resend before doing anything with this message
		if !mgr.Db().HaveISeenIt(peerMessage.MsgId) {
			// mark the sender as "seen" for this message
			mgr.Db().PeerNodeForId(from.ID().String()).AddTx(peerMessage.MsgId)
			// call the registered callback handler
			mgr.callback(from.Name(), peerMessage.MsgText)
			// forward to all other peers
			mgr.Broadcast(peerMessage)
		} else {
			// noop: discard previously seen message
		}
	}
	return err
}

func (mgr *PagerProtocolManager) Protocol() p2p.Protocol {
	proto := p2p.Protocol {
			Name:		ProtocolName,
			Version:		ProtocolVersion,
			Length:		ProtocolMsgCount,
			Run:		func(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
				mgr.logger.Debug("Connecting with '%s' [%s]", peer.Name(), peer.RemoteAddr())

				if mgr.PeerCount() >= maxPeers {
					mgr.logger.Info("Max peer capacity, cannot accept '%s'", peer.Name())
					return protocol.NewProtocolError(protocol.ErrorMaxPeersReached, "already connected to maximum peer capacity")
				}

				// initiate handshake with the new peer
				if err := mgr.Handshake(peer, &defaultHandshake, ws); err != nil {
					mgr.logger.Error("%s: %s", peer.Name(), err)
					return err
				} else {
					defer func() {
						mgr.logger.Debug("Disconnecting from '%s'", peer.Name())
						mgr.Db().UnRegisterPeerNodeForId(peer.ID().String())
						mgr.DecrPeer()
					}()
				}
				
				mgr.logger.Debug("Handshake Succeeded with '%s'", peer.Name())
				// start the listener for this node
				for {
					msg, err := ws.ReadMsg()
					if err != nil {
						return err
					}
					switch msg.Code {
						case BroadcastText:
							// handle the broadcast message
							if err := mgr.handleBroadcastMsg(msg, peer); err != nil {
								return err
							}
						default:
							// error condition, unknown protocol message
							return protocol.NewProtocolError(protocol.ErrorUnknownMessageType, "unknown protocol message recieved")
					}
				}
			},
	}
	return proto
}