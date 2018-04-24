package network

import (
	"sync"
	"fmt"
	"time"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/db"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/discover"
	ethLog "github.com/ethereum/go-ethereum/log"
)

// a trustee app for trust node mining award management
// we are abstracting the mining award management into an "app" instead of
// directly implementing it as part of network protocol layer, so that
// app can be extended to add token management APIs (e.g. balance query, balance transfer, etc)

type Trustee interface {
	NewMiningRewardTx(block consensus.Block) *consensus.Transaction
	VerifyMiningRewardTx(block consensus.Block) bool
	MiningRewardBalance(block consensus.Block, miner []byte) uint64
}

type PlatformManager interface {
	// start the platform block producer
	Start() error
	// stop the platform processing
	Stop() error
	// suspend the platform block producer
	Suspend() error
	// query status of a submitted transaction, by its transaction ID
	// returns the block where it was finalized, or error if not finalized
	Status(txId *core.Byte64) (consensus.Block, error)
	// get a snapshot of current world state
	State() *State
	// get reference to Trustee app for the stack
	Trustee() Trustee
	// submit a transaction payload, and get a transaction ID
	Submit(txPayload, signature, submitter []byte) *core.Byte64
	// get a list of current peers
	Peers() []AppConfig
	// disconnect a specific peer
	Disconnect(app *AppConfig) error
}

var (
	// size of transaction queue
	txQueueSize = 10
	// max transactions in a block
	maxTxCount = 10
	// maximum number of peer applications to connect with
	maxPeerCount = 10
	// maximum number of blocks to request at a time
	maxBlocks = 10
	// max time to wait for a transaction
	maxTxWaitSec = 10 * time.Second
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
	shutdownBlockProducer chan bool
	peerDb db.PeerSetDb
	deadTxs map[core.Byte64]bool
	peers map[string]AppConfig
	peerCount	int
	isRunning bool
}

//func NewPlatformManager(config *NetworkConfig, appDb db.Database) (*platformManager, error) {
func NewPlatformManager(appConfig *AppConfig, srvConfig *ServiceConfig, appDb db.Database) (*platformManager, error) {
	if appConfig == nil || srvConfig == nil {
		log.AppLogger().Error("incorrect or nil config")
		return nil, core.NewCoreError(ERR_INVALID_ARG, "incorrect or nil config")
	}
	if srvConfig.TxProcessor == nil || srvConfig.PeerValidator == nil {
		log.AppLogger().Error("mandatory callback not provided")
		return nil, core.NewCoreError(ERR_INVALID_ARG, "callback not provided")
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
		shutdownBlockProducer: make(chan bool),
		peerDb: db.NewPeerSetDbInMemory(),
		deadTxs: make(map[core.Byte64]bool),
		peers: make(map[string]AppConfig),
	}
	mgr.logger = log.NewLogger(*mgr)

	// configure p2p server instance
	if err := mgr.configureP2P(); err != nil {
		mgr.logger.Error("Failed to create p2p server: %s", err.Error())
		return nil, err
	}

	// TODO: change blockchain intialization to use genesis block header as parameters instead of hard coded genesis time
    if engine, err := consensus.NewBlockChainConsensus(genesisTimeStamp, mgr.config.minerId, appDb); err != nil {
		mgr.logger.Error("Failed to create consensus engine: %s", err.Error())
		return nil, err
    } else {
	    	mgr.engine = engine
	    	mgr.config.genesis = engine.Genesis()
    }    
	mgr.logger.Debug("Created new instance of counter protocol manager")
	return mgr, nil
}

func (mgr *platformManager) Start() error {
	if mgr.isRunning {
		mgr.logger.Debug("Block production already running: %s", mgr.config.NodeId)
		return core.NewCoreError(ERR_DUPLICATE_START, "block production already running")
	}

	// start block producer
	go mgr.blockProducer()
	mgr.isRunning = true
	return nil
}

func (mgr *platformManager) Suspend() error {
	if !mgr.isRunning {
		mgr.logger.Debug("Block production already suspended: %s", mgr.config.NodeId)
		return core.NewCoreError(ERR_DUPLICATE_SUSPEND, "block production already suspended")
	}

	// stop the block producer
	mgr.shutdownBlockProducer <- true
	mgr.isRunning = false
	return nil
}

func (mgr *platformManager) Stop() error {
	// stop the block producer
	mgr.shutdownBlockProducer <- true
	// stop p2p server
	mgr.srv.Stop()
	mgr.logger.Debug("Stopped p2p server")
	mgr.isRunning = false
	// stop DB
	if err := mgr.stateDb.Close(); err != nil {
		mgr.logger.Error("Failed to close state DB: %s", err)
		return err
	}
	mgr.logger.Debug("Stopped application peer node: %s", mgr.config.NodeId)
	return nil
}

func (mgr *platformManager) Status(txId *core.Byte64) (consensus.Block, error) {
	if found, _ := mgr.deadTxs[*txId]; found {
		// forget about this tx, to save memory
		delete(mgr.deadTxs, *txId)
		return nil, core.NewCoreError(consensus.ERR_TX_NOT_APPLIED, "transaction rejected")
	} else {
		return mgr.engine.TransactionStatus(txId)
	}
}

func (mgr *platformManager) State() *State {
	return &State{
		block: mgr.engine.BestBlock(),
	}
}

func (mgr *platformManager) Trustee() Trustee {
	return nil
}

func (mgr *platformManager) Submit(txPayload, signature, submitter []byte) *core.Byte64 {
	// create an instance of transaction
	tx := consensus.NewTransaction(txPayload, signature, submitter)
	// put transaction into the queue
	mgr.txQ <- tx
	// return the transaction ID for status check by application
	return tx.Id()
}

func (mgr *platformManager) Peers() []AppConfig {
	peers := make([]AppConfig,len(mgr.peers))
	i := 0
	for _, peer := range mgr.peers {
		peers[i] = peer
		i++
	}
	return peers
}

func (mgr *platformManager) Disconnect(app *AppConfig) error {
	var err error
	id := fmt.Sprintf("%x", app.NodeId)
	if node := mgr.peerDb.PeerNodeForId(id); node != nil {
		err = mgr.peerDb.UnRegisterPeerNodeForId(id)
		node.(*peerNode).Peer().Disconnect(p2p.DiscQuitting)
	} else {
		err = core.NewCoreError(ErrorNotFound, "peer not found")
	}
	return err
}

func (mgr *platformManager) configureP2P() error {
	var natAny nat.Interface
	if mgr.config.Nat {
		natAny = nat.Any()
	} else {
		natAny = nil
	}
	bootstrapNodes := make([]*discover.Node, 0, len(mgr.config.BootstrapNodes))
	for _, bootnode := range mgr.config.BootstrapNodes {
		if enode, err := discover.ParseNode(bootnode); err == nil {
			bootstrapNodes = append(bootstrapNodes, enode)
		} else {
			mgr.logger.Error("Failed to create bootnode: %s", err)
			return err
		}
	}
	// instantiate devP2P server
	serverConfig := p2p.Config{
		MaxPeers:   maxPeerCount,
		PrivateKey: mgr.config.IdentityKey,
		Name:       mgr.config.NodeName,
		ListenAddr: ":" + mgr.config.Port,
		NAT: 		natAny,
		Protocols:  mgr.protocol(),
		BootstrapNodes: bootstrapNodes,
		Logger: ethLog.Root(),
	}
	handler := ethLog.NewGlogHandler(ethLog.StreamHandler(log.GetLogFile(),ethLog.LogfmtFormat()))
	handler.Verbosity(ethLog.LvlInfo)
	serverConfig.Logger.SetHandler(handler)
	serverConfig.Logger.Info("This is a test log message using eth logger....")
	mgr.srv = &p2p.Server{Config: serverConfig}
	if err := mgr.srv.Start(); err != nil {
		mgr.logger.Error("Failed to start p2p server: %s", err)
		return err
	}
	// initialize self ID
	mgr.config.NodeId = mgr.srv.Self().ID.Bytes()
	mgr.config.minerId = mgr.srv.Self().ID.Bytes()
	mgr.logger.Debug("Started application node: %s", mgr.config.NodeId)
	return nil
}

func (mgr *platformManager) protocol() []p2p.Protocol {
	return []p2p.Protocol {p2p.Protocol {
			Name:		mgr.config.ProtocolName,
			Version:		mgr.config.ProtocolVersion,
			Length:		ProtocolMsgCount,
			Run:		func(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
				mgr.logger.Debug("Connecting with '%s' [%s]", peer.Name(), peer.RemoteAddr())
				node := NewPeerNode(peer, ws)
				
				// initiate handshake with the new peer
				myBest := mgr.engine.BestBlock()
				if err := mgr.handshake(mgr.getHandshakeMsg(myBest), node); err != nil {
					mgr.logger.Error("%s: %s", peer.Name(), err)
					return err
				} else {
					defer func() {
						mgr.logger.Debug("Disconnecting from '%s'", peer.Name())
						mgr.unregisterPeer(node)
						close(node.GetBlockHashesChan)
						close(node.GetBlocksChan)
					}()
				}
				
				mgr.logger.Debug("Handshake Succeeded with '%s'", peer.Name())

				// check and perform a sync based on total weight
				go func() {
					if err := mgr.syncNode(myBest, node); err != nil {
						mgr.logger.Error("Sync failed: '%s'", err)
					}
				}()
				
				// start the listener for this node
				if err := mgr.listen(node); err != nil {
					mgr.logger.Error("Disconnecting node: %s, Reason: %s", node.Id() ,err.Error())
				}
				return nil
			},
	}}
}

func (mgr *platformManager) broadcast(block consensus.Block) int {
	count := 0
	for _, node := range mgr.peerDb.PeerNodesWithMsgNotSeen(block.Hash()) {
		peer, _ := node.(*peerNode)
		// first mark this peer as has seen this message, so we can stop cyclic receive immediately
		peer.AddTx(block.Hash())
		peer.Send(NewBlock, block.Spec())
		mgr.logger.Debug("relayed message '%x' to %s", block.Hash(), peer.Name())
		count++
	}
	return count
}

func isSyncNeeded(best consensus.Block, node *peerNode) bool {
	return node.Status().TotalWeight.Uint64() > best.Weight().Uint64() ||
		(node.Status().TotalWeight.Uint64() == best.Weight().Uint64() &&
			node.Status().TipNumeric.Uint64() < best.Numeric())
}

func (mgr *platformManager) syncNode(best consensus.Block, node *peerNode) error {
	// check if need sync
	mgr.logger.Debug("sync: peer Weight/Numeric '%d'/'%d' vs our Weight/Numeric '%d'/'%d'",
		node.Status().TotalWeight.Uint64(), node.Status().TipNumeric.Uint64(), best.Weight().Uint64(), best.Numeric())
//	if !isSyncNeeded(best, node) {
//		return nil
//	}
	// lets assume our tip is the last known to us block on main blockchain
	node.LastHash = best.Hash()
	// wait until sync completes, or an error
	for isSyncNeeded(best, node) {
		// suspend block production
		if mgr.isRunning {
			 mgr.Suspend()
		}
		// request sync starting from the genesis block (in case there was a fork with better chain)
		if err := node.Send(GetBlockHashesRequest, GetBlockHashesRequestMsg{
				ParentHash: *node.LastHash,
				MaxBlocks: *core.Uint64ToByte8(uint64(maxBlocks)),
		}); err != nil {
			return core.NewCoreError(ErrorSyncFailed, err.Error())
		}
		// we wait for the response
		switch <- node.GetBlockHashesChan {
			case CHAN_NEXT:
				// continue to next batch
//				continue
			case CHAN_ERROR:
				return core.NewCoreError(ErrorSyncFailed, "error processing sync protocol")
			case CHAN_ABORT:
				return core.NewCoreError(ErrorSyncFailed, "sync protocol aborted")
			case CHAN_RETRY:
				// simply continue and retry
//				continue
			case CHAN_DONE:
				// signal that we are done
//				mgr.logger.Debug("Syncing with peer '%s' done", node.Id())
				return nil
		}
		best = mgr.engine.BestBlock()
//		if !mgr.isInShutdown {
//			best = mgr.engine.BestBlock()
//		}
	}
	// resume block production
	if !mgr.isRunning {
		 mgr.Start()
	}
	mgr.logger.Debug("Syncing with peer '%s' done", node.Id())
	return nil
}

func (mgr *platformManager) getHandshakeMsg(best consensus.Block) *HandshakeMsg {
	return &HandshakeMsg{
	    NetworkId: mgr.config.NetworkId,
	    Genesis: *mgr.config.genesis,
		ProtocolId: mgr.config.ProtocolId,
		TotalWeight: *best.Weight(),
		TipNumeric: *core.Uint64ToByte8(best.Numeric()),
	}
}

func (mgr *platformManager) listen(peer *peerNode) error {
	for {
		msg, err := peer.ReadMsg()
		if err != nil {
			peer.Peer().Log().Debug("Error: %s", err.Error())
			return err
		}
		switch msg.Code {
			case GetBlockHashesRequest:
				// handle the sync request message to fetch hashes
				if err := mgr.handleGetBlockHashesRequestMsg(msg, peer); err != nil {
					return err
				}
			case GetBlockHashesResponse:
				// handle the sync response message with hashes
				if err := mgr.handleGetBlockHashesResponseMsg(msg, peer); err != nil {
					return err
				}
			case GetBlockHashesRewind:
				// handle the sync rewind message
				if err := mgr.handleGetBlockHashesRewindMsg(msg, peer); err != nil {
					return err
				}
			case GetBlocksRequest:
				// handle sync request to fetch block specs
				if err := mgr.handleGetBlocksRequestMsg(msg, peer); err != nil {
					return err
				}
			case GetBlocksResponse:
				// handle sync request to fetch block specs
				if err := mgr.handleGetBlocksResponseMsg(msg, peer); err != nil {
					return err
				}
			case NewBlock:
				// handle new block announcement
				if err := mgr.handleNewBlockMsg(msg, peer); err != nil {
					return err
				}
			default:
				// error condition, unknown protocol message
				err := core.NewCoreError(ErrorUnknownMessageType, "unknown protocol message recieved")
				return err
		}
	}
}

func (mgr *platformManager) handleGetBlockHashesRewindMsg(msg p2p.Msg, from *peerNode) error {
	// find an ancestor to current tip and retry sync
	if from.LastHash == nil {
		from.LastHash = mgr.engine.BestBlock().Hash()
	}
	// walk back 10 ancestors and retry
	if ancestor, err := mgr.engine.Ancestor(from.LastHash, 10); err != nil {
		mgr.logger.Error("Failed to rewind back from hash '%x'", *from.LastHash)
		return err
	} else {
		from.LastHash = ancestor.Hash()
	}
	mgr.logger.Debug("Rewinding back to hash '%x'", *from.LastHash)
	from.GetBlockHashesChan <- CHAN_RETRY
	return nil
}

func (mgr *platformManager) processBlock(block consensus.Block, from *peerNode) error {
	// process block transactions
	for _,tx := range block.Transactions() {
		// use application callback to process transaction
		if !mgr.processTx(&tx, block) {
			// there was some problem during processing the transaction
			return core.NewCoreError(consensus.ERR_INVALID_TX, "transaction error")
		}
	}
	// submit block for acceptance
	if err := mgr.engine.AcceptNetworkBlock(block); err != nil {
		return err
	} else {
		// mark the sender has having seen this message
		from.AddTx(block.Hash())
		// check if we have seen this before
		if mgr.peerDb.HaveISeenIt(*block.Hash()) {
			mgr.logger.Debug("dropping duplicate forwarded message: %x", *block.Hash())
			return core.NewCoreError(consensus.ERR_DUPLICATE_BLOCK, "repeated block")
		}
		// broadcast block to peers
		mgr.logger.Debug("Forwarding accepted network block from %s", from.Id())
		mgr.broadcast(block)
	}
	return nil
}

func (mgr *platformManager) handleNewBlockMsg(msg p2p.Msg, from *peerNode) error {
	if block, err := mgr.engine.DecodeNetworkBlock(msg); err != nil {
		return core.NewCoreError(ErrorBadBlock, err.Error())
	} else {
		// process the block
		if err := mgr.processBlock(block, from); err != nil && err.(*core.CoreError).Code() != consensus.ERR_DUPLICATE_BLOCK {
			mgr.logger.Error("%s: Could not accept new block: %s", from.Id(), err)
			// abort sync
			from.GetBlockHashesChan <- CHAN_ERROR
			return err
		}
	}
	return nil	
}

func (mgr *platformManager) handleGetBlocksRequestMsg(msg p2p.Msg, to *peerNode) error {
	var request GetBlocksRequestMsg
	if err := msg.Decode(&request); err != nil {
		return core.NewCoreError(ErrorSyncFailed, err.Error())
	}
	hashes := []core.Byte64(request)
	mgr.logger.Debug("Syncing: sending '%d' blocks to '%s'", len(hashes), to.Id())
	if len(hashes) < 1 {
		mgr.logger.Error("GetBlocksRequestMsg does not have any hashes")
		return core.NewCoreError(ErrorInvalidRequest, "GetBlocksRequestMsg does not have any hashes")
	}
//	blocks := make([]*core.BlockSpec, len(hashes), len(hashes))
	blocks := make([]consensus.BlockSpec, len(hashes), len(hashes))
	i := 0
	for _, hash := range hashes {
		if block, err := mgr.engine.Block(&hash); err == nil {
			blocks[i] = block.Spec()
			mgr.logger.Debug("Syncing: sending block: %x", *block.Hash())
			i++
		}
//		if blockNode, found := mgr.chain.BlockNode(&hash); found {
//			if blocks[i], found = mgr.chain.BlockSpec(blockNode.Block()); found {
//				i++
//			}
//		}
	}
	if i < 1 {
		// no blocks to send
		mgr.logger.Debug("GetBlocksRequestMsg did not find any blocks")
		return core.NewCoreError(ErrorNotFound, "GetBlocksRequestMsg did not find any blocks")
	}
	// trim return list of blocks to match correct size
	blocks = blocks[:i]

	// send back the blocks in response
	if err := to.Send(GetBlocksResponse, GetBlocksResponseMsg(blocks)); err != nil {
		mgr.logger.Debug("failed to send GetBlocksResponse")
		return err
	}
	return nil
}

func (mgr *platformManager) handleGetBlocksResponseMsg(msg p2p.Msg, from *peerNode) error {
	// read the response message
	var response GetBlocksResponseMsg
	
	if err := msg.Decode(&response); err != nil {
		from.GetBlockHashesChan <- CHAN_ERROR
		return core.NewCoreError(ErrorSyncFailed, err.Error())
	}
//	specs := []*core.BlockSpec(response)
	specs := []consensus.BlockSpec(response)
	if len(specs) < 1 {
		mgr.logger.Error("GetBlocksResponseMsg does not have any blocks from '%s'", from.Id())
		return core.NewCoreError(ErrorInvalidRequest, "GetBlocksResponseMsg does not have any blocks")
	}
//	defer mgr.saveState()

	// walk through the list of blocks and add them into our blockchain
	for _, spec := range specs {
		// process the new block, and ignore duplicate add errors
//		if delta, hash, err := mgr.processBlockSpec(spec, from); err != nil && err.(*core.CoreError).Code() != core.ERR_DUPLICATE_BLOCK {
		var block consensus.Block
		var err error
		if block, err = mgr.engine.DecodeNetworkBlockSpec(spec); err != nil {
			mgr.logger.Error("Failed to decode network block spec: %s", err)
			return err
		}
		if err := mgr.processBlock(block, from); err != nil && err.(*core.CoreError).Code() != consensus.ERR_DUPLICATE_BLOCK {
			mgr.logger.Error("%s: Could not accept sync block: %s", from.Id(), err)
			// abort sync
			from.GetBlockHashesChan <- CHAN_ERROR
			return err
		} else {
//			// only update counter if it was not a duplicate block
//			if err == nil {
//				mgr.state.Countr += delta
//			}
			from.LastHash = block.Hash()
		}
	}
	// done processing batch of hashes, ask for next batch
	from.GetBlockHashesChan <- CHAN_NEXT
	return nil
}

func (mgr *platformManager) handleGetBlockHashesRequestMsg(msg p2p.Msg, from *peerNode) error {
	var request GetBlockHashesRequestMsg
	if err := msg.Decode(&request); err != nil {
		return core.NewCoreError(ErrorSyncFailed, err.Error())
	}
	mgr.logger.Debug("Syncing: request to fetch '%d' hashes after '%x'", request.MaxBlocks.Uint64(), request.ParentHash)
	// fetch hashes from blockchain DB
//	blocks := mgr.chain.Blocks(&request.ParentHash, request.MaxBlocks.Uint64())
	blocks, _ := mgr.engine.Descendents(&request.ParentHash, int(request.MaxBlocks.Uint64()))
//	if err != nil {
//		mgr.logger.Error("failed to fetch descendents: %s", err)
//		return err
//	}
	mgr.logger.Debug("Syncing: sending '%d' blocks to '%s'", len(blocks), from.Id())
	if len(blocks) < 1 {
		// no blocks to send
		mgr.logger.Debug("%s: GetBlockHashesRequest does not have any hashes", from.Id())
		// notify peer to rewind and re-sync from start
//		if err := from.Send(GetBlockHashesRewind, GetBlockHashesRewindMsg(*mgr.engine.Genesis().Hash())); err != nil {
		if err := from.Send(GetBlockHashesRewind, GetBlockHashesRewindMsg(*core.BytesToByte64(nil))); err != nil {
			mgr.logger.Debug("failed to send GetBlockHashesRewind")
			return err
		}
		return nil
	}
	hashes := make([]core.Byte64, len(blocks), len(blocks))
	for i, block := range blocks {
		hashes[i] = *block.Hash()
		mgr.logger.Debug("Adding hash to response: %x", hashes[i])
	}
	// send back the hashes in response
	if err := from.Send(GetBlockHashesResponse, GetBlockHashesResponseMsg(hashes)); err != nil {
		mgr.logger.Debug("failed to send GetBlockHashesResponse")
		return err
	}
	return nil
}

func (mgr *platformManager) handleGetBlockHashesResponseMsg(msg p2p.Msg, from *peerNode) error {
	// read the response message
	var response GetBlockHashesResponseMsg
	
	if err := msg.Decode(&response); err != nil {
		from.GetBlockHashesChan <- CHAN_ERROR
		return core.NewCoreError(ErrorSyncFailed, err.Error())
	}
	hashes := []core.Byte64(response)
	mgr.logger.Debug("Syncing: received '%d' hashes from '%s'", len(hashes), from.Id())
	for _, hash := range hashes {
		mgr.logger.Debug("Syncing: received hash: %x", hash)
	}
	if len(hashes) < 1 {
		// no hashesh to fetch
		from.GetBlockHashesChan <- CHAN_DONE
		return nil
	}
	// walk through the hashes in the response and ask for blocks
	// (what about if a hash has already been fetched from another peer during concurrent sync?)
	// (in that case, our DB update will simply skip duplicate entry)
	mgr.logger.Debug("Syncing: requesting '%d' blocks from '%s'", len(hashes), from.Id())
	if err := from.Send(GetBlocksRequest, GetBlocksRequestMsg(hashes)); err!= nil {
		from.GetBlockHashesChan <- CHAN_ABORT
		return core.NewCoreError(ErrorSyncFailed, err.Error())
	}
	
	return nil
}

func (mgr *platformManager) PeerCount() int {
	return mgr.peerCount
}

func (mgr *platformManager) registerPeer(peer PeerNode, handshake *HandshakeMsg, peerConf *AppConfig) error {
	if err := mgr.peerDb.RegisterPeerNode(peer); err != nil {
		return err
	} else {
		mgr.peerCount++
		peer.SetStatus(handshake)
		mgr.peers[peer.Id()] = *peerConf
	}
	return nil
}

func (mgr *platformManager) unregisterPeer(node PeerNode) {
	mgr.peerDb.UnRegisterPeerNodeForId(node.Id())
	mgr.peerCount--
	delete(mgr.peers, node.Id())
}

//func (mgr *platformManager) AddPeer(node *discover.Node) error {
//	// we don't have a p2p server for individual protocol manager, and hence cannot add a node
//	// this will need to be done from outside, at the application level
//	return core.NewCoreError(ErrorNotImplemented, "protocol manager cannot add peer")
//}

// perform sub protocol handshake
func (mgr *platformManager) handshake(status *HandshakeMsg, peer PeerNode) error {
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
		case handshake.Genesis != *mgr.config.genesis:
			mgr.logger.Error("genesis does not match")
			return core.NewCoreError(ErrorHandshakeFailed, "genesis does not match")
	}

	// validate peer connection with application
	peerConf := AppConfig {
		// identified application's shard/group on public p2p network
		NetworkId: handshake.NetworkId,
		// peer's node ID, extracted from p2p connection request
		NodeId:	peer.NodeId(),
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
	return mgr.registerPeer(peer, handshake, &peerConf)
}

func (mgr *platformManager) mineCandidateBlock(newBlock consensus.Block) bool {
	done := make (chan struct{})
	success := false
	mgr.logger.Debug("sending block for mining")
	mgr.engine.MineCandidateBlockPoW(newBlock, consensus.PowApprover(mgr.config.PowApprover), func(block consensus.Block, err error) {
			if err != nil {
				mgr.logger.Debug("failed to mine candidate block: %s", err)
			} else {
				mgr.logger.Debug("successfully mined candidate block")
				// send the block over the wire
				mgr.broadcast(newBlock)
				success = true
			}
			done <- struct{}{}
	})
	// wait for mining to finish
	mgr.logger.Debug("waiting for mining to complete")
	<- done	
	return success
}

// background go routine to produce blocks with transactions from queue
func (mgr *platformManager) blockProducer() {
	mgr.logger.Debug("starting up block producer")
	var newBlock consensus.Block
//	newBlock := mgr.engine.NewCandidateBlock()
	txCount := 0
	// start a timer, in case there are no transactions
	mgr.logger.Debug("starting timer for %d duration", maxTxWaitSec)
	wait := time.NewTimer(maxTxWaitSec)
	defer wait.Stop()
	for {
		select {
			case <- mgr.shutdownBlockProducer:
				mgr.logger.Debug("shutting down block producer")
				return
			case tx := <- mgr.txQ:
				if newBlock == nil {
					newBlock = mgr.engine.NewCandidateBlock()
				}
				mgr.logger.Debug("processing transaction id: %x", *tx.Id())
				if mgr.processTx(tx, newBlock) {
					// application processed transaction successfully, add to block
					newBlock.AddTransaction(tx)
					txCount++
					if txCount == maxTxCount {
						// we are at capacity, no more transactions in this block
						mgr.logger.Debug("reached maximum transactions for current block")
						wait.Reset(maxTxWaitSec*10)
						if !mgr.mineCandidateBlock(newBlock) {
							mgr.logger.Debug("block failed to mine")
							// add all transactions of the block to dead list
							for _,tx := range newBlock.Transactions() {
								mgr.deadTxs[*tx.Id()] = true
							}
						}
						newBlock = nil
						txCount = 0
						wait.Reset(maxTxWaitSec)
					} else {
						// we got at least one transaction, so no need to wait
						wait.Reset(50 * time.Millisecond)
					}
				} else {
					mgr.logger.Debug("application rejected transaction: %x", *tx.Id())
					mgr.deadTxs[*tx.Id()] = true
				}
			case <- wait.C:
				if newBlock == nil {
					newBlock = mgr.engine.NewCandidateBlock()
				}
				mgr.logger.Debug("timed out waiting for transactions")
				wait.Reset(maxTxWaitSec*10)
				if !mgr.mineCandidateBlock(newBlock) {
					mgr.logger.Debug("block failed to mine")
					// add all transactions of the block to dead list
					for _,tx := range newBlock.Transactions() {
						mgr.deadTxs[*tx.Id()] = true
					}
				}
				newBlock = nil
				txCount = 0
				wait.Reset(maxTxWaitSec)
		}
	}
}

// should be called by the background block producer go routine
func (mgr *platformManager) processTx(tx *consensus.Transaction, block consensus.Block) bool {
	// validate signature of the transaction
	//
	// Actually, different applications will have different signature scheme, and hence
	// we'll need to rely on application to implement the appropriate and correct signature
	// and verification scheme
	return mgr.config.TxProcessor(&Transaction{
		Transaction: *tx,
		block: block,
	})
}
