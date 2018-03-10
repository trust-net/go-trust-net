package counter

import (
//	"math/big"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/config"
	"github.com/trust-net/go-trust-net/core/chain"
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
)

// supported versions of the protocol for this codebase
var ProtocolVersion = uint(poc1)

var handshakeMsg = protocol.HandshakeMsg {
	NetworkId: *core.BytesToByte16([]byte{1,2,3,4}),
	ShardId: *core.BytesToByte16(nil),
	TotalWeight: *core.Uint64ToByte8(0),
}

var networkCountr = []byte("WorldState-Countr")

type worldState struct {
	Countr int64
}

// a "countr" protocol manager implementation
type CountrProtocolManager struct {
	protocol.ManagerBase
	logger log.Logger
	state worldState
	chain *chain.BlockChainInMem
	genesis *core.SimpleBlock
	miner *core.SimpleNodeInfo
	stateDb db.Database
}

// create a new instance of countr protocol manager
func NewCountrProtocolManager(miner string) *CountrProtocolManager {
	mgr := CountrProtocolManager{
		state: worldState{0},
		miner: core.NewSimpleNodeInfo(miner),
		genesis: core.NewSimpleBlock(core.BytesToByte64(nil), 0, 0, genesisTimeStamp, core.NewSimpleNodeInfo("")),
	}
	mgr.logger = log.NewLogger(mgr)
	mgr.logger.Debug("Created new instance of counter protocol manager")
	mgr.genesis.ComputeHash()
	config, _ := config.Config()
	mgr.stateDb = config.Db()
	// initialize the world state from DB, if exists
	if data, err := mgr.stateDb.Get(networkCountr); err == nil {
		if err := common.Deserialize(data, &mgr.state); err != nil {
			mgr.logger.Error("Failed to initialize world state from DB: %s", err.Error())
			return nil
		} else {
			mgr.logger.Debug("Initialized world state from DB: %s", mgr.state)
		}
	} else {
		mgr.logger.Debug("Using blank world state: %s", mgr.state)
	}
	if chain, err := chain.NewBlockChainInMem(mgr.genesis, config.Db()); err != nil {
		mgr.logger.Error("Failed to create blockchain: %s", err.Error())
		return nil
	} else {
		mgr.chain = chain
	}
	mgr.SetDb(db.NewPeerSetDbInMemory())
	return &mgr
}

func (mgr *CountrProtocolManager) saveState() error {
	if data, err := common.Serialize(&mgr.state); err != nil {
		return protocol.NewProtocolError(protocol.ERR_WORLD_STATE_UPDATE_FAILURE, err.Error())
	} else {
		if err := mgr.stateDb.Put(networkCountr, data); err != nil {
			return protocol.NewProtocolError(protocol.ERR_WORLD_STATE_UPDATE_FAILURE, err.Error())
		}
	}
	return nil
}

func (mgr *CountrProtocolManager) Countr() int64 {
	return mgr.state.Countr
}

func (mgr *CountrProtocolManager) Shutdown() {
	if err := mgr.saveState(); err != nil {
		mgr.logger.Error("Failed to cleanly shutdown chain DB: %s", err.Error())
	}
	if err := mgr.chain.Shutdown(); err != nil {
		mgr.logger.Error("Failed to cleanly shutdown chain DB: %s", err.Error())
	}
	mgr.stateDb.Close()
	mgr.logger.Debug("shutting down counter protocol manager")
}

func (mgr *CountrProtocolManager) delta(opCode *core.Byte8) bool {
	// create new block and add to my blockchain
	block := core.NewSimpleBlock(mgr.chain.Tip().Hash(), mgr.chain.Tip().Weight()+1, mgr.chain.Tip().Depth()+1, 0, mgr.miner)
	block.AddTransaction(opCode)
	block.ComputeHash()
	if err := mgr.chain.AddBlockNode(block); err != nil {
		mgr.logger.Error("Failed to increment counter: %s", err.Error())
		return false
	}
	// broadcast counter change to peers
	count := mgr.broadCast(block)
	mgr.logger.Debug("Relayed new block to %d peers", count)
	return true
}

func (mgr *CountrProtocolManager) Increment(delta int) {
	mgr.logger.Debug("Incrementing network counter from '%d' --> '%d'", mgr.state.Countr, mgr.state.Countr+int64(delta))
	defer mgr.saveState()
	for delta > 0 {
		if  mgr.delta(OpIncrement) {
			// increment our counter
			mgr.state.Countr++
			delta--
		} else {
			return
		}
	}
}

func (mgr *CountrProtocolManager) Decrement(delta int) {
	mgr.logger.Debug("Decrementing network counter from '%d' --> '%d'", mgr.state.Countr, mgr.state.Countr-int64(delta))
	defer mgr.saveState()
	for delta > 0 {
		if  mgr.delta(OpDecrement) {
			// increment our counter
			mgr.state.Countr--
			delta--
		} else {
			return
		}
	}
}

func (mgr *CountrProtocolManager) broadCast(block core.Block) int {
	count := 0
	for _, node := range mgr.Db().PeerNodesWithMsgNotSeen(block.Hash()) {
		peer, _ := node.(*protocol.Node)
		// first mark this peer as has seen this message, so we can stop cyclic receive immediately
		peer.AddTx(block.Hash())
		peer.Send(NewBlock, NewBlockMsg(*core.NewBlockSpecFromBlock(block)))
		mgr.logger.Debug("relayed message '%x' to %s", block.Hash(), peer.Peer().Name())
		count++
	}
	return count
}

func (mgr *CountrProtocolManager) getHandshakeMsg() *protocol.HandshakeMsg {
	handshakeMsg.TotalWeight = *core.Uint64ToByte8(mgr.chain.Depth())
	return &handshakeMsg
}

// this log needs to be revisited, we need to better handle two cases:
//    #1 when there was a fork and alternate best chain, we need our world state to be re-adjusted due to fork
//    #2 when we sync after a restart, we need to skip the hashes already known, and start asking only unknown blocks 
func (mgr *CountrProtocolManager) syncNode(node *protocol.Node) error {
	// check if need sync
	if node.Status().TotalWeight.Uint64() <= mgr.getHandshakeMsg().TotalWeight.Uint64() {
		return nil
	}
	// lets assume our tip is the last known to us block on main blockchain
	node.LastHash = mgr.chain.Tip().Hash()
	// wait until sync completes, or an error
	for node.Status().TotalWeight.Uint64() > mgr.getHandshakeMsg().TotalWeight.Uint64() {
		mgr.logger.Debug("Requesting sync: peer weight '%d' > our weight '%d'",
			node.Status().TotalWeight.Uint64(), mgr.getHandshakeMsg().TotalWeight.Uint64())
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
				continue
			case protocol.CHAN_ERROR:
				return protocol.NewProtocolError(protocol.ErrorSyncFailed, "error processing sync protocol")
			case protocol.CHAN_ABORT:
				return protocol.NewProtocolError(protocol.ErrorSyncFailed, "sync protocol aborted")
			case protocol.CHAN_RETRY:
				// simply continue and retry
				continue
			case protocol.CHAN_DONE:
				// signal that we are done
				mgr.logger.Debug("Syncing with peer '%s' done", node.ID())
				return nil
		}
		
	}
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
					mgr.logger.Error("Error: %s", err.Error())
					return err
				}
			case GetBlockHashesResponse:
				// handle the sync response message with hashes
				if err := mgr.handleGetBlockHashesResponseMsg(msg, peer); err != nil {
					mgr.logger.Error("Error: %s", err.Error())
					return err
				}
			case GetBlockHashesRewind:
				// handle the sync rewind message
				if err := mgr.handleGetBlockHashesRewindMsg(msg, peer); err != nil {
					mgr.logger.Error("Error: %s", err.Error())
					return err
				}
			case GetBlocksRequest:
				// handle sync request to fetch block specs
				if err := mgr.handleGetBlocksRequestMsg(msg, peer); err != nil {
					mgr.logger.Error("Error: %s", err.Error())
					return err
				}
			case GetBlocksResponse:
				// handle sync request to fetch block specs
				if err := mgr.handleGetBlocksResponseMsg(msg, peer); err != nil {
					mgr.logger.Error("Error: %s", err.Error())
					return err
				}
			case NewBlock:
				// handle new block announcement
				if err := mgr.handleNewBlockMsg(msg, peer); err != nil {
					peer.Peer().Log().Error("Error: %s", err.Error())
					return err
				}
			default:
				// error condition, unknown protocol message
				err := protocol.NewProtocolError(protocol.ErrorUnknownMessageType, "unknown protocol message recieved")
				mgr.logger.Error("Error: %s", err.Error())
				return err
		}
	}
}

func (mgr *CountrProtocolManager) processBlockSpec(spec *core.BlockSpec, from *protocol.Node) (int64, *core.Byte64, error) {
		block := core.NewSimpleBlockFromSpec(spec)
		var delta int64
		switch block.OpCode().Uint64() {
			case opIncrement:
				delta = 1
			case opDecrement:
				delta = -1
			default:
				mgr.logger.Error("Invalid opcode '%d' from '%s'", block.OpCode().Uint64(), from.ID())
				return 0, nil, protocol.NewProtocolError(protocol.ErrorInvalidResponse, "GetBlocksResponseMsg has invalid opcode")
		}
		// add block to our blockchain
		if err := mgr.chain.AddBlockNode(block); err != nil {
			mgr.logger.Error("Failed to add new block from '%s'", from.ID())
			return delta, block.Hash(), err
		}
		// update our counter
		return delta, block.Hash(), nil	
}

func (mgr *CountrProtocolManager) handleGetBlockHashesRewindMsg(msg p2p.Msg, from *protocol.Node) error {
	var rewindHash GetBlockHashesRewindMsg
	if err := msg.Decode(&rewindHash); err != nil {
		return protocol.NewProtocolError(protocol.ErrorInvalidResponse, err.Error())
	}
	// validate that specified hash is in our DB
	hash := core.Byte64(rewindHash)
	if blockNode, found := mgr.chain.BlockNode(&hash); !found {
		// peer tried to misdirect us to invalid hash
		mgr.logger.Debug("Invalid rewind hash '%x' from '%s'", hash, from.ID())
		from.GetBlockHashesChan <- protocol.CHAN_ABORT
		return protocol.NewProtocolError(protocol.ErrorInvalidResponse, "invalid hash in rewind msg")
	} else {
		// rewind to specified hash and restart sync
		mgr.logger.Debug("Rewinding back to hash '%x' as suggested by '%s'", hash, from.ID())
		from.LastHash = blockNode.Hash()
		// ideally we want to reset world state to the world state corresponding to block node
		// but for POC Iteration 1 we are just going back to genesis and restarting
		if *blockNode.Hash() != *mgr.genesis.Hash() {
			mgr.logger.Debug("Rewind hash provided does not match genesis, from '%s'", from.ID())
			from.GetBlockHashesChan <- protocol.CHAN_ABORT
			return protocol.NewProtocolError(protocol.ErrorInvalidResponse, "rewind hash does not match genesis")
		}
		mgr.state.Countr = 0
		mgr.saveState()
		mgr.chain.Flush()
		from.GetBlockHashesChan <- protocol.CHAN_RETRY
	}
	return nil
}

func (mgr *CountrProtocolManager) handleNewBlockMsg(msg p2p.Msg, from *protocol.Node) error {
	var newBlockMsg NewBlockMsg
	if err := msg.Decode(&newBlockMsg); err != nil {
		return protocol.NewProtocolError(protocol.ErrorBadBlock, err.Error())
	}
	// process the block
	spec := core.BlockSpec(newBlockMsg)
	if delta, hash, err := mgr.processBlockSpec(&spec, from); err != nil {
		// abort sync
		from.GetBlockHashesChan <- protocol.CHAN_ERROR
		return err
	} else {
		mgr.state.Countr += delta
		from.LastHash = hash
		mgr.saveState()
	}
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
	blocks := make([]*core.BlockSpec, len(hashes), len(hashes))
	i := 0
	for _, hash := range hashes {
		if blockNode, found := mgr.chain.BlockNode(&hash); found {
			if blocks[i], found = mgr.chain.BlockSpec(blockNode.Block()); found {
				i++
			}
		}
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
	specs := []*core.BlockSpec(response)
	if len(specs) < 1 {
		mgr.logger.Error("GetBlocksResponseMsg does not have any blocks from '%s'", from.ID())
		return protocol.NewProtocolError(protocol.ErrorInvalidRequest, "GetBlocksResponseMsg does not have any blocks")
	}
	defer mgr.saveState()

	// walk through the list of blocks and add them into our blockchain
	for _, spec := range specs {
		// process the new block, and ignore duplicate add errors
		if delta, hash, err := mgr.processBlockSpec(spec, from); err != nil && err.(*core.CoreError).Code() != core.ERR_DUPLICATE_BLOCK {
			// abort sync
			from.GetBlockHashesChan <- protocol.CHAN_ERROR
			return err
		} else {
			// only update counter if it was not a duplicate block
			if err == nil {
				mgr.state.Countr += delta
			}
			from.LastHash = hash
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
	blocks := mgr.chain.Blocks(&request.ParentHash, request.MaxBlocks.Uint64())
	mgr.logger.Debug("Syncing: sending '%d' blocks to '%s'", len(blocks), from.ID())
	if len(blocks) < 1 {
		// no blocks to send
		mgr.logger.Debug("%s: GetBlockHashesRequest does not have any hashes", from.ID())
		// notify peer to rewind and re-sync from start
		if err := from.Send(GetBlockHashesRewind, GetBlockHashesRewindMsg(*mgr.genesis.Hash())); err != nil {
			mgr.logger.Debug("failed to send GetBlockHashesRewind")
			return err
		}
		return nil
	}
	hashes := make([]core.Byte64, len(blocks), len(blocks))
	for i, block := range blocks {
		hashes[i] = *block.Hash()
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
				if err := mgr.Handshake(mgr.getHandshakeMsg(), node); err != nil {
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
					if err := mgr.syncNode(node); err != nil {
						mgr.logger.Error("Sync failed: '%s'", err)
					}
				}()
				
				// start the listener for this node
				mgr.listen(node)
				return nil
			},
	}
	return proto
}
