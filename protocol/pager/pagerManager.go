package pager

import (
	"time"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/trust-net/go-trust-net/protocol"
)

const (
	// maximum number of peers to connect at any given time
	maxPeers = 10
)

// short protocol name for handshake negotiation
var ProtocolName = "pager"

// known protocol versions
const (
	poc1 = 0x01
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

var DefaultHandshake = protocol.HandshakeMsg {
	NetworkId: *protocol.BytesToByte16([]byte{1,2,3,4}),
}

// signature for callback function when a page is recieved
type PageHandler func (from, txt string)

// a "byoi" protocol manager implementation
type PagerProtocolManager struct {
	db db.PeerSetDb
	peerCount	int
	logger log.Logger
	callback PageHandler
}

// create a new instance of pager protocol manager
func NewPagerProtocolManager(callback PageHandler) *PagerProtocolManager {
	mgr := PagerProtocolManager{
		db: db.NewPeerSetDbInMemory(),
		peerCount: 0,
		callback: callback,
	}
	mgr.logger = log.NewLogger(mgr)
	return &mgr
}

func (mgr *PagerProtocolManager) Db() db.PeerSetDb {
	return mgr.db
}

func (mgr *PagerProtocolManager) AddPeer(node *discover.Node) error {
	// we don't have a p2p server for individual protocol manager, and hence cannot add a node
	// this will need to be done from outside, at the application level
	return protocol.NewProtocolError(protocol.ErrorNotImplemented, "protocol manager cannot add peer")
}

func (mgr *PagerProtocolManager) Handshake(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
	// send our status to the peer
	if err := p2p.Send(ws, protocol.Handshake, DefaultHandshake); err != nil {
		return protocol.NewProtocolError(protocol.ErrorHandshakeFailed, err.Error())
	}

	// wait on peer's status for upto 5 seconds
	wait := time.NewTimer(5 * time.Second)
	defer wait.Stop()
	var msg p2p.Msg
	var err error
	done := make(chan int)
	go func(){
		var err error
		if msg, err = ws.ReadMsg(); err == nil {
			done <- 1
		}
	}()
	select {
		case <- done:
			break
		case <- wait.C:
			err = protocol.NewProtocolError(protocol.ErrorHandshakeFailed, "timed out waiting for handshake status")
	}
	if err != nil {
		return err
	}
	
	// make sure its a handshake status message
	if msg.Code != protocol.Handshake {
		return protocol.NewProtocolError(protocol.ErrorHandshakeFailed, "first message needs to be handshake status")
	}
	var handshake protocol.HandshakeMsg
	err = msg.Decode(&handshake)
	if err != nil {
		return protocol.NewProtocolError(protocol.ErrorHandshakeFailed, err.Error())
	}
	
	// validate handshake message
	switch {
		case handshake.NetworkId != DefaultHandshake.NetworkId:
			return protocol.NewProtocolError(protocol.ErrorHandshakeFailed, "network ID does not match")
		case handshake.ShardId != DefaultHandshake.ShardId:
			return protocol.NewProtocolError(protocol.ErrorHandshakeFailed, "shard ID does not match")
	}

	// add the peer into our DB
	node := protocol.NewNode(peer, ws)
	if err = mgr.db.RegisterPeerNode(node); err != nil {
		return err
	} else {
		mgr.peerCount++
		node.SetStatus(&handshake)
	}
	return nil
}

func (mgr *PagerProtocolManager) Broadcast(msg BroadcastTextMsg) int {
	count := 0
	for _, node := range mgr.db.PeerNodesWithMsgNotSeen(msg.MsgId) {
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
		if !mgr.db.HaveISeenIt(peerMessage.MsgId) {
			// mark the sender as "seen" for this message
			mgr.db.PeerNodeForId(from.ID().String()).AddTx(peerMessage.MsgId)
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

				if mgr.peerCount >= maxPeers {
					mgr.logger.Info("Max peer capacity, cannot accept '%s'", peer.Name())
					return protocol.NewProtocolError(protocol.ErrorMaxPeersReached, "already connected to maximum peer capacity")
				}

				// initiate handshake with the new peer
				if err := mgr.Handshake(peer, ws); err != nil {
					mgr.logger.Error("%s", err)
					return err
				} else {
					defer func() {
						mgr.logger.Debug("Disconnecting from '%s'", peer.Name())
						mgr.db.UnRegisterPeerNodeForId(peer.ID().String())
						mgr.peerCount--
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