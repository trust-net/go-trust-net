package network

import (
	"sync"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type PlatformManager interface {
	// submit a transaction payload, and get a transaction ID
	Submit(txPayload []byte, submitter *core.Byte64) *core.Byte64
	// query status of a submitted transaction, by its transaction ID
	// returns the block where it was finalized, or error if not finalized
	Status(txId *core.Byte64) (consensus.Block, error)
	// get a list of current peers
	Peers() []AppConfig
	// disconnect a specific peer
	Disconnect(app *AppConfig) error
	// start the platform processing
	Start() error
	// stop the platform processing
	Stop() error
}

var (
	// size of transaction queue
	txQueueSize = 10
	// wait limit for handshake message read
	msgReadTimeout = 5
	// start time for genesis block
	genesisTimeStamp = uint64(0x200000)
)

type platformManager struct {
	config *PlatformConfig
	lock sync.RWMutex
	srv *p2p.Server
	logger log.Logger
	engine consensus.Consensus
	stateDb db.Database
	txQ chan *consensus.Transaction
	peerDb db.PeerSetDb
	peerCount	int
}

//func NewPlatformManager(config *NetworkConfig, appDb db.Database) (*platformManager, error) {
func NewPlatformManager(appConfig *AppConfig, srvConfig *ServiceConfig, appDb db.Database) (*platformManager, error) {
	if appConfig == nil || srvConfig == nil {
		log.AppLogger().Error("incorrect or nil config")
		return nil, core.NewCoreError(ERR_INVALID_ARG, "incorrect or nil config")
	}
	config := &PlatformConfig {
		AppConfig: *appConfig,
		ServiceConfig: *srvConfig,
	}
	if appDb == nil {
		log.AppLogger().Error("incorrect or nil app DB")
		return nil, core.NewCoreError(ERR_INVALID_ARG, "incorrect or nil app DB")
	}
	mgr := &platformManager {
		config: config,
		stateDb: appDb,
		txQ: make(chan *consensus.Transaction, txQueueSize),
		peerDb: db.NewPeerSetDbInMemory(),
	}
	// TODO: change blockchain intialization to use genesis block header as parameters instead of hard coded genesis time
    if engine, err := consensus.NewBlockChainConsensus(genesisTimeStamp, &mgr.config.MinerId, appDb); err != nil {
		mgr.logger.Error("Failed to create consensus engine: %s", err.Error())
		return nil, err
    } else {
	    	mgr.engine = engine
	    	mgr.config.genesis = *engine.Genesis()
    }
    
// Below needs to be done as part of start up
//	// instantiate devP2P server
//	serverConfig := p2p.Config{
//		MaxPeers:   10,
//		PrivateKey: mgr.config.IdentityKey,
//		Name:       mgr.config.NodeName,
//		ListenAddr: ":" + mgr.config.Port,
////		NAT: 		mgr.config.Nat,
//		Protocols:  protocols,
//		BootstrapNodes: config.Bootnodes(),
//	}
//	srv := &p2p.Server{Config: serverConfig}
	mgr.logger = log.NewLogger(mgr)
	mgr.logger.Debug("Created new instance of counter protocol manager")
	return mgr, nil
}


func (mgr *platformManager) PeerCount() int {
	return mgr.peerCount
}

func (mgr *platformManager) UnregisterPeer(node PeerNode) {
	mgr.peerDb.UnRegisterPeerNodeForId(node.Id())
	mgr.peerCount--
}

func (mgr *platformManager) AddPeer(node *discover.Node) error {
	// we don't have a p2p server for individual protocol manager, and hence cannot add a node
	// this will need to be done from outside, at the application level
	return core.NewCoreError(ErrorNotImplemented, "protocol manager cannot add peer")
}

// perform sub protocol handshake
func (mgr *platformManager) Handshake(status *HandshakeMsg, peer PeerNode) error {
	// send our status to the peer
	if err := peer.Send(Handshake, *status); err != nil {
		return core.NewCoreError(ErrorHandshakeFailed, err.Error())
	}

	var msg p2p.Msg
	var err error
	err = common.RunTimeBoundSec(msgReadTimeout, func() error {
			msg, err = peer.ReadMsg()
			return err
		}, core.NewCoreError(ErrorHandshakeFailed, "timed out waiting for handshake status"))
	if err != nil {
		return err
	}

	// make sure its a handshake status message
	if msg.Code != Handshake {
		return core.NewCoreError(ErrorHandshakeFailed, "first message needs to be handshake status")
	}
	var handshake HandshakeMsg
	err = msg.Decode(&handshake)
	if err != nil {
		return core.NewCoreError(ErrorHandshakeFailed, err.Error())
	}
	return mgr.validateAndAdd(&handshake, peer)
}

// validate hanshake message from peer and add to peer set
func (mgr *platformManager) validateAndAdd(handshake *HandshakeMsg, peer PeerNode) error {
	// validate handshake message
	switch {
		case handshake.NetworkId != mgr.config.NetworkId:
			mgr.logger.Error("network ID does not match")
			return core.NewCoreError(ErrorHandshakeFailed, "network ID does not match")
		case handshake.Genesis != mgr.config.genesis:
			mgr.logger.Error("genesis does not match")
			return core.NewCoreError(ErrorHandshakeFailed, "genesis does not match")
	}

	// validate peer connection with application
	peerConf := AppConfig {
		// identified application's shard/group on public p2p network
		NetworkId: handshake.NetworkId,
		// peer's node ID, extracted from p2p connection request
		MinerId:	peer.NodeId(),
		// peer node's name
		NodeName: peer.Name(),
		// application's protocol
		ProtocolId: handshake.ProtocolId,
//		// application's authentication/authorization token
//		AuthToken: handshake.AuthToken,
	}
	if err := mgr.config.PeerValidator(&peerConf); err != nil {
		// application's peer validation failed
		mgr.logger.Error("Peer application failed validation: %s", err)
		return err
	}
	// add the peer into our DB
	if err := mgr.peerDb.RegisterPeerNode(peer); err != nil {
		return err
	} else {
		mgr.peerCount++
		peer.SetStatus(&peerConf)
	}
	return nil
}
