package counter

import (
//	"math/big"
//    "sync"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
//	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/trust-net/go-trust-net/config"
//	"github.com/trust-net/go-trust-net/core/chain"
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
	// start time for genesis block
	genesisTimeStamp = 0x200000
	// network countr name
	countrName = "WorldState-Countr"
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

var handshakeMsg = protocol.HandshakeMsg {
	NetworkId: *core.BytesToByte16([]byte{1,2,3,4}),
	ShardId: *core.BytesToByte16(nil),
	TotalWeight: *core.Uint64ToByte8(0),
}

// a "countr" protocol manager implementation
type CountrProtocolManager struct {
	protocol.ManagerBase
	logger log.Logger
//	state worldState
//	chain *chain.BlockChainInMem
//	genesis *core.SimpleBlock
	engine consensus.Consensus
//	miner *core.SimpleNodeInfo
	miner *core.Byte64
	stateDb db.Database
	isInShutdown bool
//	lock sync.RWMutex
//	nodes map[*protocol.Node]bool
}

// create a new instance of countr protocol manager
func NewCountrProtocolManager(miner string) *CountrProtocolManager {
	mgr := CountrProtocolManager{
		miner: core.BytesToByte64([]byte(miner)),
		isInShutdown: false,
//		nodes: make(map[*protocol.Node]bool),
//		genesis: core.NewSimpleBlock(core.BytesToByte64(nil), 0, 0, genesisTimeStamp, core.NewSimpleNodeInfo("")),
	}
	mgr.logger = log.NewLogger(mgr)
	mgr.logger.Debug("Created new instance of counter protocol manager")
//	mgr.genesis.ComputeHash()
	config, _ := config.Config()
	mgr.stateDb = config.Db()
//	// initialize the world state from DB, if exists
//	if data, err := mgr.stateDb.Get(networkCountr); err == nil {
//		if err := common.Deserialize(data, &mgr.state); err != nil {
//			mgr.logger.Error("Failed to initialize world state from DB: %s", err.Error())
//			return nil
//		} else {
//			mgr.logger.Debug("Initialized world state from DB: %s", mgr.state)
//		}
//	} else {
//		mgr.logger.Debug("Using blank world state: %s", mgr.state)
//	}
//	if chain, err := chain.NewBlockChainInMem(mgr.genesis, config.Db()); err != nil {
//		mgr.logger.Error("Failed to create blockchain: %s", err.Error())
//		return nil
//	} else {
//		mgr.chain = chain
//	}
    if engine, err := consensus.NewBlockChainConsensus(genesisTimeStamp, mgr.miner, config.Db()); err != nil {
		mgr.logger.Error("Failed to create blockchain: %s", err.Error())
		return nil
    } else {
	    	mgr.engine = engine
	    	handshakeMsg.Genesis = *engine.Genesis()
    }
	mgr.SetDb(db.NewPeerSetDbInMemory())
	return &mgr
}

//func (mgr *CountrProtocolManager) saveState() error {
//	if data, err := common.Serialize(&mgr.state); err != nil {
//		return protocol.NewProtocolError(protocol.ERR_WORLD_STATE_UPDATE_FAILURE, err.Error())
//	} else {
//		if err := mgr.stateDb.Put(networkCountr, data); err != nil {
//			return protocol.NewProtocolError(protocol.ERR_WORLD_STATE_UPDATE_FAILURE, err.Error())
//		}
//	}
//	return nil
//}

func (mgr *CountrProtocolManager) Countr(name string) uint64 {
	countr,_ := mgr.countr(name, mgr.engine.BestBlock())
	return countr
}

func (mgr *CountrProtocolManager) countr(name string, block consensus.Block) (uint64, error) {
	if value, err := block.Lookup([]byte(name)); err != nil {
		return 0, err
	} else {
		return core.BytesToByte8(value).Uint64(), nil
	}
}

func (mgr *CountrProtocolManager) Shutdown() {
//	if err := mgr.saveState(); err != nil {
//		mgr.logger.Error("Failed to cleanly shutdown chain DB: %s", err.Error())
//	}
//	if err := mgr.chain.Shutdown(); err != nil {
//		mgr.logger.Error("Failed to cleanly shutdown chain DB: %s", err.Error())
//	}
//    // diconnect with all peers
//	mgr.lock.Lock()
//    for node,_ := range mgr.nodes {
//	    	node.Peer().Disconnect(p2p.DiscQuitting)
//    }
//	mgr.lock.Unlock()
	mgr.isInShutdown = true
	mgr.stateDb.Close()
	mgr.logger.Debug("shutting down counter protocol manager")
}

func (mgr *CountrProtocolManager) Tip() (uint64, uint64, *core.Byte64) {
	bb := mgr.engine.BestBlock()
	mgr.logger.Debug("Current tip of the blockchain: Depth %d, Weight %d\n%x", bb.Depth().Uint64(), bb.Weight().Uint64(), *bb.Hash())
	return bb.Depth().Uint64(), bb.Weight().Uint64(), bb.Hash()
}

func makeTransactionPayload(name string, opCode *core.Byte8) []byte {
	data := make([]byte, 0, len(name)+8)
	data = append(data, opCode.Bytes()...)
	data = append(data, []byte(name)...)
	return data
} 

func getTransactionFromPayload(data []byte) (string, *core.Byte8) {
	if len(data) < 9 {
		return "", nil
	}
	return string(data[8:]), core.BytesToByte8(data[:8])	
} 

func (mgr *CountrProtocolManager) delta(name string, opCode *core.Byte8, delta int) bool {
	// create new block and add to my blockchain
	block := mgr.engine.NewCandidateBlock()
//	current, _ := mgr.countr(name, block)
//	if err != nil {
//		mgr.logger.Error("Failed to get current countr value: %s", err)
//		// using default value
////		return false
//	}
	for ;delta > 0; delta-- {
		current, _ := mgr.countr(name, block)
		switch *opCode {
			case *OpIncrement:
				block.Update([]byte(name), core.Uint64ToByte8(current+1).Bytes())
			case *OpDecrement:
				block.Update([]byte(name), core.Uint64ToByte8(current-1).Bytes())
		}
		block.AddTransaction(consensus.NewTransaction(makeTransactionPayload(name, opCode), mgr.miner))
	}
	// submit block for mining
	done := make(chan struct{})
	result := false
	mgr.engine.MineCandidateBlock(block, func(block consensus.Block, err error) {
			result = err == nil
			defer func() {
				mgr.logger.Debug("about to send signal from callback")
				done <- struct{}{}
				mgr.logger.Debug("done sending signal from callback")
			}()
			if err != nil {
				mgr.logger.Error("failed to mine candidate block: %s", err)
				return
			}
			// broadcast the block to network
			count := mgr.broadCast(block)
			mgr.logger.Debug("Relayed new block to %d peers", count)
	});
	// wait for our callback to finish
	mgr.logger.Debug("waiting for signal from callback")
	<-done
	mgr.logger.Debug("recieved signal from callback")
	return result
}

func (mgr *CountrProtocolManager) Increment(name string, delta int) (uint64) {
	for delta > 0 {
		if  mgr.delta(name, OpIncrement, 1) {
			delta--
		} else {
			return 0
		}
	}
//	mgr.delta(name, OpIncrement, delta)
	return mgr.Countr(countrName)
}

func (mgr *CountrProtocolManager) Decrement(name string, delta int) (uint64) {
	for delta > 0 {
		if  mgr.delta(name, OpDecrement, 1) {
			delta--
		} else {
			return 0
		}
	}
//	mgr.delta(name, OpIncrement, delta)
	return mgr.Countr(countrName)
}

func (mgr *CountrProtocolManager) broadCast(block consensus.Block) int {
	count := 0
	for _, node := range mgr.Db().PeerNodesWithMsgNotSeen(block.Hash()) {
		peer, _ := node.(*protocol.Node)
		// first mark this peer as has seen this message, so we can stop cyclic receive immediately
		peer.AddTx(block.Hash())
		peer.Send(NewBlock, block.Spec())
		mgr.logger.Debug("relayed message '%x' to %s", block.Hash(), peer.Peer().Name())
		count++
	}
	return count
}

func (mgr *CountrProtocolManager) getHandshakeMsg(best consensus.Block) *protocol.HandshakeMsg {
	handshakeMsg.TotalWeight = *best.Weight()
	handshakeMsg.TipNumeric = *core.Uint64ToByte8(best.Numeric())
	return &handshakeMsg
}

func isSyncNeeded(best consensus.Block, node *protocol.Node) bool {
	return node.Status().TotalWeight.Uint64() < best.Weight().Uint64() ||
		(node.Status().TotalWeight.Uint64() == best.Weight().Uint64() &&
			node.Status().TipNumeric.Uint64() > best.Numeric())
}

func (mgr *CountrProtocolManager) syncNode(best consensus.Block, node *protocol.Node) error {
	// check if need sync
	mgr.logger.Debug("sync: peer Weight/Numeric '%d'/'%d' vs our Weight/Numeric '%d'/'%d'",
		node.Status().TotalWeight.Uint64(), node.Status().TipNumeric.Uint64(), best.Weight().Uint64(), best.Numeric())
	if !isSyncNeeded(best, node) {
		return nil
	}
	// lets assume our tip is the last known to us block on main blockchain
	node.LastHash = best.Hash()
	// wait until sync completes, or an error
	for !mgr.isInShutdown && isSyncNeeded(best, node) {
		// request sync starting from the genesis block (in case there was a fork with better chain)
		if err := node.Send(GetBlockHashesRequest, GetBlockHashesRequestMsg{
				ParentHash: *node.LastHash,
				MaxBlocks: *core.Uint64ToByte8(uint64(maxBlocks)),
		}); err != nil {
			return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
		}
		// we wait for the response
		switch <- node.GetBlockHashesChan {
			case protocol.CHAN_NEXT:
				// continue to next batch
//				continue
			case protocol.CHAN_ERROR:
				return protocol.NewProtocolError(protocol.ErrorSyncFailed, "error processing sync protocol")
			case protocol.CHAN_ABORT:
				return protocol.NewProtocolError(protocol.ErrorSyncFailed, "sync protocol aborted")
			case protocol.CHAN_RETRY:
				// simply continue and retry
//				continue
			case protocol.CHAN_DONE:
				// signal that we are done
//				mgr.logger.Debug("Syncing with peer '%s' done", node.ID())
				return nil
		}
		if !mgr.isInShutdown {
			best = mgr.engine.BestBlock()
		}
	}
	mgr.logger.Debug("Syncing with peer '%s' done", node.ID())
	return nil
}

func (mgr *CountrProtocolManager) listen(peer *protocol.Node) error {
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
				err := protocol.NewProtocolError(protocol.ErrorUnknownMessageType, "unknown protocol message recieved")
				return err
		}
	}
}

//func (mgr *CountrProtocolManager) processBlockSpec(spec *core.BlockSpec, from *protocol.Node) (int64, *core.Byte64, error) {
//		block := core.NewSimpleBlockFromSpec(spec)
//		var delta int64
//		switch block.Transactions()[0].Uint64() {
//			case opIncrement:
//				delta = 1
//			case opDecrement:
//				delta = -1
//			default:
//				mgr.logger.Error("Invalid opcode '%d' from '%s'", block.Transactions()[0].Uint64(), from.ID())
//				return 0, nil, protocol.NewProtocolError(protocol.ErrorInvalidResponse, "GetBlocksResponseMsg has invalid opcode")
//		}
//		// add network block to our blockchain
//		if err := mgr.chain.AddNetworkNode(block); err != nil {
//			mgr.logger.Error("Failed to add new block from '%s'", from.ID())
//			return delta, block.Hash(), err
//		}
//		// mark the sender has having seen this message
//		from.AddTx(block.Hash())
//		// broadcast message to other peers
//		mgr.broadCast(block)
//		// update our counter
//		return delta, block.Hash(), nil	
//}

func (mgr *CountrProtocolManager) handleGetBlockHashesRewindMsg(msg p2p.Msg, from *protocol.Node) error {
	// this message simply means our node needs to be manually restarted with clean slate
	mgr.logger.Error("manual flush needed due to rewind message recieved from '%s'", from.ID())
	return protocol.NewProtocolError(protocol.ErrorInvalidResponse, "need manual rewind")
//	var rewindHash GetBlockHashesRewindMsg
//	if err := msg.Decode(&rewindHash); err != nil {
//		return protocol.NewProtocolError(protocol.ErrorInvalidResponse, err.Error())
//	}
//	// validate that specified hash is in our DB
//	hash := core.Byte64(rewindHash)
//	if blockNode, found := mgr.chain.BlockNode(&hash); !found {
//		// peer tried to misdirect us to invalid hash
//		mgr.logger.Debug("Invalid rewind hash '%x' from '%s'", hash, from.ID())
//		from.GetBlockHashesChan <- protocol.CHAN_ABORT
//		return protocol.NewProtocolError(protocol.ErrorInvalidResponse, "invalid hash in rewind msg")
//	} else {
//		// rewind to specified hash and restart sync
//		mgr.logger.Debug("Rewinding back to hash '%x' as suggested by '%s'", hash, from.ID())
//		from.LastHash = blockNode.Hash()
//		// ideally we want to reset world state to the world state corresponding to block node
//		// but for POC Iteration 1 we are just going back to genesis and restarting
//		if *blockNode.Hash() != *mgr.genesis.Hash() {
//			mgr.logger.Debug("Rewind hash provided does not match genesis, from '%s'", from.ID())
//			from.GetBlockHashesChan <- protocol.CHAN_ABORT
//			return protocol.NewProtocolError(protocol.ErrorInvalidResponse, "rewind hash does not match genesis")
//		}
//		mgr.state.Countr = 0
//		mgr.saveState()
//		mgr.chain.Flush()
//		from.GetBlockHashesChan <- protocol.CHAN_RETRY
//	}
//	return nil
}

func (mgr *CountrProtocolManager) processBlock(block consensus.Block, from *protocol.Node) error {
	// process block transactions
	for _,tx := range block.Transactions() {
		result := false
		name, opCode := getTransactionFromPayload(tx.Payload)
		if opCode == nil {
			mgr.logger.Error("Invalid transaction: %s", tx)
			return core.NewCoreError(consensus.ERR_INVALID_TX, "transaction error")
		}
		current, _ := mgr.countr(name, block)
//		if err != nil {
//			mgr.logger.Error("Failed to get current countr value: %s", err)
//			// using default value
//	//		return false
//		}
		switch *opCode {
			case *OpIncrement:
				result = block.Update([]byte(name), core.Uint64ToByte8(current+1).Bytes())
			case *OpDecrement:
				result = block.Update([]byte(name), core.Uint64ToByte8(current-1).Bytes())
			default:
				mgr.logger.Error("Invalid opcode: %s", *opCode)
		}
		if !result {
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
		// broadcast block to peers
		mgr.logger.Debug("Forwarding accepted network block from %s", from.ID())
		mgr.broadCast(block)
	}
	return nil
}

func (mgr *CountrProtocolManager) handleNewBlockMsg(msg p2p.Msg, from *protocol.Node) error {
//	var newBlockMsg NewBlockMsg
//	if err := msg.Decode(&newBlockMsg); err != nil {
//		return protocol.NewProtocolError(protocol.ErrorBadBlock, err.Error())
//	}
	if block, err := mgr.engine.DecodeNetworkBlock(msg); err != nil {
		return protocol.NewProtocolError(protocol.ErrorBadBlock, err.Error())
	} else {
		// process the block
		if err := mgr.processBlock(block, from); err != nil && err.(*core.CoreError).Code() != consensus.ERR_DUPLICATE_BLOCK {
			mgr.logger.Error("%s: Could not accept new block: %s", from.ID(), err)
			// abort sync
			from.GetBlockHashesChan <- protocol.CHAN_ERROR
			return err
		}
	}
//	// process the block
//	spec := core.BlockSpec(newBlockMsg)
//	if delta, hash, err := mgr.processBlockSpec(&spec, from); err != nil {
//		// abort sync
//		from.GetBlockHashesChan <- protocol.CHAN_ERROR
//		return err
//	} else {
//		mgr.state.Countr += delta
//		from.LastHash = hash
//		mgr.saveState()
//	}
	return nil	
}

func (mgr *CountrProtocolManager) handleGetBlocksRequestMsg(msg p2p.Msg, to *protocol.Node) error {
	var request GetBlocksRequestMsg
	if err := msg.Decode(&request); err != nil {
		return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
	}
	hashes := []core.Byte64(request)
	mgr.logger.Debug("Syncing: sending '%d' blocks to '%s'", len(hashes), to.ID())
	if len(hashes) < 1 {
		mgr.logger.Error("GetBlocksRequestMsg does not have any hashes")
		return protocol.NewProtocolError(protocol.ErrorInvalidRequest, "GetBlocksRequestMsg does not have any hashes")
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
		return protocol.NewProtocolError(protocol.ErrorNotFound, "GetBlocksRequestMsg did not find any blocks")
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

func (mgr *CountrProtocolManager) handleGetBlocksResponseMsg(msg p2p.Msg, from *protocol.Node) error {
	// read the response message
	var response GetBlocksResponseMsg
	
	if err := msg.Decode(&response); err != nil {
		from.GetBlockHashesChan <- protocol.CHAN_ERROR
		return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
	}
//	specs := []*core.BlockSpec(response)
	specs := []consensus.BlockSpec(response)
	if len(specs) < 1 {
		mgr.logger.Error("GetBlocksResponseMsg does not have any blocks from '%s'", from.ID())
		return protocol.NewProtocolError(protocol.ErrorInvalidRequest, "GetBlocksResponseMsg does not have any blocks")
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
			mgr.logger.Error("%s: Could not accept sync block: %s", from.ID(), err)
			// abort sync
			from.GetBlockHashesChan <- protocol.CHAN_ERROR
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
	from.GetBlockHashesChan <- protocol.CHAN_NEXT
	return nil
}

func (mgr *CountrProtocolManager) handleGetBlockHashesRequestMsg(msg p2p.Msg, from *protocol.Node) error {
	var request GetBlockHashesRequestMsg
	if err := msg.Decode(&request); err != nil {
		return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
	}
	mgr.logger.Debug("Syncing: request to fetch '%d' hashes after '%x'", request.MaxBlocks.Uint64(), request.ParentHash)
	// fetch hashes from blockchain DB
//	blocks := mgr.chain.Blocks(&request.ParentHash, request.MaxBlocks.Uint64())
	blocks, err := mgr.engine.Descendents(&request.ParentHash, int(request.MaxBlocks.Uint64()))
	if err != nil {
		mgr.logger.Error("failed to fetch descendents: %s", err)
		return err
	}
	mgr.logger.Debug("Syncing: sending '%d' blocks to '%s'", len(blocks), from.ID())
	if len(blocks) < 1 {
		// no blocks to send
		mgr.logger.Debug("%s: GetBlockHashesRequest does not have any hashes", from.ID())
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

func (mgr *CountrProtocolManager) handleGetBlockHashesResponseMsg(msg p2p.Msg, from *protocol.Node) error {
	// read the response message
	var response GetBlockHashesResponseMsg
	
	if err := msg.Decode(&response); err != nil {
		from.GetBlockHashesChan <- protocol.CHAN_ERROR
		return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
	}
	hashes := []core.Byte64(response)
	mgr.logger.Debug("Syncing: received '%d' hashes from '%s'", len(hashes), from.ID())
	for _, hash := range hashes {
		mgr.logger.Debug("Syncing: received hash: %x", hash)
	}
	if len(hashes) < 1 {
		// no hashesh to fetch
		from.GetBlockHashesChan <- protocol.CHAN_DONE
		return nil
	}
	// walk through the hashes in the response and ask for blocks
	// (what about if a hash has already been fetched from another peer during concurrent sync?)
	// (in that case, our DB update will simply skip duplicate entry)
	mgr.logger.Debug("Syncing: requesting '%d' blocks from '%s'", len(hashes), from.ID())
	if err := from.Send(GetBlocksRequest, GetBlocksRequestMsg(hashes)); err!= nil {
		from.GetBlockHashesChan <- protocol.CHAN_ABORT
		return protocol.NewProtocolError(protocol.ErrorSyncFailed, err.Error())
	}
	
	return nil
}

func (mgr *CountrProtocolManager) Protocol() p2p.Protocol {
	proto := p2p.Protocol {
			Name:		ProtocolName,
			Version:		ProtocolVersion,
			Length:		ProtocolMsgCount,
			Run:		func(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
				mgr.logger.Debug("Connecting with '%s' [%s]", peer.Name(), peer.RemoteAddr())
				node := protocol.NewNode(peer, ws)
				
				// initiate handshake with the new peer
				myBest := mgr.engine.BestBlock()
				if err := mgr.Handshake(mgr.getHandshakeMsg(myBest), node); err != nil {
					mgr.logger.Error("%s: %s", peer.Name(), err)
					return err
				} else {
					defer func() {
						mgr.logger.Debug("Disconnecting from '%s'", peer.Name())
						mgr.UnregisterPeer(node)
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
	}
	return proto
}
