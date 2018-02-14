package counter

import (
	"math/big"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/go-trust-net/protocol"
)

// short protocol name for handshake negotiation
var ProtocolName = "countr"

const (
	// current protocol version
	poc1 = 0x01
	// maximum number of blocks to request at a time
	maxBlocks = 10
	// maximum sync wait time
	maxSyncWait = 300
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

var handshakeMsg = protocol.HandshakeMsg {
	NetworkId: *protocol.BytesToByte16([]byte{1,2,3,4}),
	TD: big.NewInt(0),
}

// a "countr" protocol manager implementation
type CountrProtocolManager struct {
	protocol.ManagerBase
	logger log.Logger
	count int64
}

// create a new instance of countr protocol manager
func NewCountrProtocolManager() *CountrProtocolManager {
	mgr := CountrProtocolManager{
		count: 0,
	}
	mgr.SetDb(db.NewPeerSetDbInMemory())
	mgr.SetHandshakeMsg(&handshakeMsg)
	mgr.logger = log.NewLogger(mgr)
	mgr.logger.Debug("Created new instance of counter protocol manager")
	return &mgr
}

func (mgr *CountrProtocolManager) syncNode(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
	// first check if our total difficulty is worse than the peer
	node := mgr.Db().PeerNodeForId(peer.ID().String()).(*protocol.Node)

	// wait until sync completes, or an error
	for node.Status().TD.Cmp(handshakeMsg.TD) > 0 {
		mgr.logger.Debug("Requesting sync: peer diffculty '%d' > our difficulty '%d'", node.Status().TD, handshakeMsg.TD)
		// request sync starting with current status
		if err := p2p.Send(ws, SyncRequest, SyncRequestMsg{
				StartHash: handshakeMsg.CurrentBlock.Bytes(),
				MaxBlocks: big.NewInt(maxBlocks),
		}); err != nil {
			return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
		}	
	}
	return nil
}

func (mgr *CountrProtocolManager) listen(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
	for {
		msg, err := ws.ReadMsg()
		if err != nil {
			return err
		}
		switch msg.Code {
			case SyncResponse:
				// handle the sync response message
				if err := mgr.handleSyncResponseMsg(msg, peer); err != nil {
					return err
				}
			default:
				// error condition, unknown protocol message
				return protocol.NewProtocolError(protocol.ErrorUnknownMessageType, "unknown protocol message recieved")
		}
	}
}

func (mgr *CountrProtocolManager) handleSyncResponseMsg(msg p2p.Msg, from *p2p.Peer) error {
	// TODO
	mgr.logger.Info("Need to implement sync response handler!")
	return nil
}
func (mgr *CountrProtocolManager) Protocol() p2p.Protocol {
	proto := p2p.Protocol {
			Name:		ProtocolName,
			Version:		ProtocolVersion,
			Length:		ProtocolMsgCount,
			Run:		func(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
				// initiate handshake with the new peer
				if err := mgr.Handshake(peer, ws); err != nil {
					mgr.logger.Error("%s", err)
					return err
				} else {
					defer func() {
						mgr.logger.Debug("Disconnecting from '%s'", peer.Name())
						mgr.Db().UnRegisterPeerNodeForId(peer.ID().String())
						mgr.DecrPeer()
					}()
				}
				
				mgr.logger.Debug("Handshake Succeeded with '%s'", peer.Name())

				// check and perform a sync based on total difficulty
				go func() {
					if err := mgr.syncNode(peer, ws); err != nil {
						mgr.logger.Error("Sync failed: '%s'", err)
					}
				}()
				
				// start the listener for this node
				mgr.listen(peer, ws)
				return nil
			},
	}
	return proto
}
