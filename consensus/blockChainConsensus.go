package consensus

import (
	"sync"
    "time"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/go-trust-net/core/trie"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/common"
)

const (
	maxBlocks = 100
	maxUncleDistance = uint64(5)
)

var tableChainNode = []byte("ChainNode-")
var tableBlock = []byte("Block-")
var dagTip = []byte("ChainState-DagTip")
var genesisParent = core.BytesToByte64(nil)
func tableKey(prefix []byte, key *core.Byte64) []byte {
	return append(prefix, key.Bytes()...)
}

type uncle struct {
	hash *core.Byte64
	miner *core.Byte64
	depth uint64
	distance uint64
}

// A blockchain based consensus platform implementation
type BlockChainConsensus struct {
	state trie.WorldState
	tip *block
	weight uint64
	genesisNode *chainNode
	minerId *core.Byte64
	db db.Database
	lock sync.RWMutex
	logger log.Logger
}

// application is responsible to create an instance of DB initialized to application's name space
func NewBlockChainConsensus(genesisTime uint64,
	minerId *core.Byte64, db db.Database) (*BlockChainConsensus, error) {
	chain := BlockChainConsensus{
		db: db,
		minerId: minerId,
		state: trie.NewMptWorldState(db),
	}
	chain.logger = log.NewLogger(chain)

	// genesis is statically defined using default values
	genesisBlock := newBlock(genesisParent, 0, 0, genesisTime, core.BytesToByte64(nil), chain.state)
	genesisBlock.computeHash()
	chain.genesisNode = newChainNode(genesisBlock)
	chain.genesisNode.setMainList(true)

	// read tip hash from DB
	if data, err := chain.db.Get(dagTip); err != nil {
		chain.logger.Debug("No DAG tip in DB, using genesis as tip")
		chain.tip = genesisBlock
		// save the tip in DB
		if err := chain.db.Put(dagTip, chain.tip.Hash().Bytes()); err != nil {
			chain.logger.Error("Failed to save DAG tip in DB: %s", err.Error())
			return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
		}
		// following is needed so that any immediate chid can be consistently operated on,
		// e.g. when traversing ancestors to find uncles
		// save the genesis block in DB
		if err := chain.putBlock(genesisBlock); err != nil {
			chain.logger.Error("Failed to save genesis block in DB: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
		if err := chain.putChainNode(chain.genesisNode); err != nil {
			chain.logger.Error("Failed to save genesis chain node in DB: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
	} else {
		hash := core.BytesToByte64(data)
		chain.logger.Debug("Found DAG tip from DB: '%x'", *hash)
		// read the tip block
		var err error
		if chain.tip, err = chain.getBlock(hash); err != nil {
			chain.logger.Error("Failed to get tip block from DB: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
		// rebase state trie to current tip
		if err = chain.state.Rebase(chain.tip.STATE); err != nil {
			chain.logger.Error("Failed to rebase world state to tip's world state: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
	}
	chain.weight = chain.tip.Weight().Uint64()
	chain.logger.Debug("Initialized block chain DAG")
	return &chain, nil
}

// return the tip of current canonical blockchain
func (c *BlockChainConsensus) Tip() Block {
	return c.tip
}


func (chain *BlockChainConsensus) getChainNode(hash *core.Byte64) (*chainNode, error) {
	if data, err := chain.db.Get(tableKey(tableChainNode, hash)); err != nil {
		chain.logger.Debug("Did not find chain node in DB: %x", *hash)
		return nil, err
	} else {
		var node chainNode
		if err := common.Deserialize(data, &node); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, err
		}
//		chain.logger.Debug("Fetched chain node from DB: %x", *hash)
		return &node, nil
	}
}

func (chain *BlockChainConsensus) getBlock(hash *core.Byte64) (*block, error) {
	if data, err := chain.db.Get(tableKey(tableBlock, hash)); err != nil {
		chain.logger.Debug("Did not find block in DB: %s", err.Error())
		return nil, err
	} else {
		var block *block
		var err error
		if block, err = deSerializeBlock(data); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, err
		}
//		chain.logger.Debug("fetched block from DB: %x", *hash)
		return block, nil
	}
}

// persist a blocknode into DB
func (chain *BlockChainConsensus) putChainNode(node *chainNode) error {
	if data, err := common.Serialize(node); err == nil {
//		chain.logger.Debug("Saved chain node in DB: %x", *node.hash())
		return chain.db.Put(tableKey(tableChainNode, node.hash()), data)
	} else {
		return err
	}
	
}

// persist a block into DB
func (chain *BlockChainConsensus) putBlock(block *block) error {
	if data, err := serializeBlock(block); err == nil {
//		chain.logger.Debug("Saved block in DB: %x", *block.Hash())
		return chain.db.Put(tableKey(tableBlock, block.Hash()), data)
	} else {
		return err
	}
}

// get a new "candidate" block, initialized with a copy of world state
// from request time tip of the canonical chain
func (c *BlockChainConsensus) NewCandidateBlock() Block {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// create a copy of world state from current canonical tip's world view
	state := trie.NewMptWorldState(c.db)
	if err := state.Rebase(c.state.Hash()); err != nil {
		c.logger.Error("failed to initialize candidate block's world state: %s", err.Error())
		return nil
	}
	
	// create a new candidate block instance initialized as child of current canonical chain tip
	b := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, state)

	// add mining reward for miner node in this block's world view
	// TODO

	// add uncles to the block
	for _, uncle := range c.findUncles(c.Tip().ParentHash(), c.Tip().Hash(), maxUncleDistance, c.Tip().Depth().Uint64()) {
//		c.logger.Debug("Adding %d distant uncle: %x, miner: %x", uncle.distance, *uncle.hash, *uncle.miner)
		b.addUncle(uncle.hash)

		// add mining reward for uncle in this block's world view
		// TODO
	}
	return b
}


func (c *BlockChainConsensus) findUncles(grandParent, parent *core.Byte64, remainingDistance, maxDepth uint64) []uncle {
	uncles := make([]uncle, 0, 5)
	if remainingDistance == 0 || *grandParent == *genesisParent {
//		c.logger.Debug("findUncles: reached max uncle search: remainingDistance %d", remainingDistance)
		return uncles
	}
	if node, err := c.getChainNode(grandParent); err == nil {
		for _, childNode := range node.children() {
			if childNode != nil && *childNode != *parent {
				if childBlock, err := c.getBlock(childNode); err == nil {
//					c.logger.Debug("findUncles: %x, remainingDistance: %d, depth %d", *childBlock.Hash(), remainingDistance, childBlock.Depth().Uint64())
					uncles = append(uncles, uncle{
							hash: childBlock.Hash(),
							miner: childBlock.Miner(),
							depth: childBlock.Depth().Uint64(),
							distance: maxUncleDistance - remainingDistance + 1,
							
					})
					uncles = append(uncles, c.findNonDirectAncestors(childNode, remainingDistance-1, maxDepth)...)
				} 
			}
		}
		uncles = append(uncles, c.findUncles(node.parent(), node.hash(), remainingDistance-1, maxDepth)...)
	}
	return uncles
}

func (c *BlockChainConsensus) findNonDirectAncestors(childNode *core.Byte64, remainingDistance, maxDepth uint64) []uncle {
	uncles := make([]uncle, 0, 5)
	if remainingDistance == 0 {
//		c.logger.Debug("findNonDirectAncestors: reached max uncle search: remainingDistance %d", remainingDistance)
		return uncles
	}
	
	if node, err := c.getChainNode(childNode); err == nil {
		for _, grandChild := range node.children() {
			if grandChild != nil {
				if grandChildBlock, err := c.getBlock(grandChild); err == nil && grandChildBlock.Depth().Uint64() <= maxDepth {
//					c.logger.Debug("findNonDirectAncestors: %x, remainingDistance: %d, depth %d", *grandChildBlock.Hash(), remainingDistance, grandChildBlock.Depth().Uint64())
					uncles = append(uncles, uncle{
							hash: grandChildBlock.Hash(),
							miner: grandChildBlock.Miner(),
							depth: grandChildBlock.Depth().Uint64(),
							distance: maxUncleDistance - remainingDistance,
							
					})
					uncles = append(uncles, c.findNonDirectAncestors(grandChild, remainingDistance-1, maxDepth)...)
				} 
			}
		}
	}
	return uncles
}

// submit a "filled" block for mining (executes as a goroutine)
// it will mine the block and update canonical chain or abort if a new network block
// is received with same or higher weight, the callback MiningResultHandler will be called
// with serialized data for the block that can be  sent over the wire to peers,
// or error if mining failed/aborted
func (c *BlockChainConsensus) MineCandidateBlock(b Block, cb MiningResultHandler) {
	go c.mineCandidateBlock(b.(*block), cb)
}
func (c *BlockChainConsensus) mineCandidateBlock(child *block, cb MiningResultHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// validate the block
	if parent, err := c.validateBlock(child); err != nil {
		cb(nil, err)
	} else {
		// mine the block
		if child.computeHash() == nil {
			c.logger.Error("Failed to compute hash for block")
			cb(nil, core.NewCoreError(ERR_BLOCK_UNHASHED, "hash computation failed"))
			return
		}
		// add the block
		if err := c.addValidatedBlock(child, parent); err != nil {
			c.logger.Debug("%s: validation failed for mined block: %x", *c.minerId, *child.Hash())
			cb(nil, err)
			return
		}
		// return raw block, so that protocol layer can update "seen" node set with hash of the block
		cb(child, nil)
		c.logger.Debug("%s: Successfully mined block: %x", *c.minerId, *child.Hash())
	}
}

// query status of a transaction (its block details) in the canonical chain
func (c *BlockChainConsensus) TransactionStatus(tx *Transaction) (Block, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	// lookup transaction in the current canonical chain
	if hash, err := c.state.HasTransaction(tx.Id()); err != nil {
		c.logger.Debug("Transaction does not exists: %x", tx.Id())
		return nil, core.NewCoreError(ERR_TX_NOT_FOUND, "transaction not found")
	} else 
	// find the block that finalized the transaction
	if block, err := c.getBlock(hash); err != nil {
		c.logger.Error("Failed to get block for transaction: %s", err)
		return nil, core.NewCoreError(ERR_DB_CORRUPTED, "error reading transaction's block")
	} else {
		return block, nil
	}
}

// validate uncle relationship
func (c *BlockChainConsensus) isUncleValid(child *block, uHash *core.Byte64) bool {
	// check if uncle is known
	if uncle, err := c.getChainNode(uHash); err != nil {
		c.logger.Error("Failed to find uncle's chain node: %s", err)
		return false
	} else {
		// find common ancestor
		distance := uint64(0)
		var parent *chainNode
		for parent, _ = c.getChainNode(&child.PHASH); parent != nil && parent.Depth > uncle.Depth; {
//			c.logger.Debug("Parent Depth: %d, Uncle Depth: %d", parent.Depth, uncle.Depth)
//			c.logger.Debug("Moving uncle up: %x --> %x", *parent.Hash, *parent.Parent)
			parent, _ = c.getChainNode(parent.Parent)
			distance++
		}
		for ; parent != nil && uncle != nil && uncle.Depth > parent.Depth; {
//			c.logger.Debug("Parent Depth: %d, Uncle Depth: %d", parent.Depth, uncle.Depth)
//			c.logger.Debug("Moving uncle up: %x --> %x", *uncle.Hash, *uncle.Parent)
			uncle, _ = c.getChainNode(uncle.Parent)
			distance++
		}
		if parent == nil || uncle == nil {
			c.logger.Error("failed to find common ancestor")
			// did not find common ancestor
			return false
		}
		if *parent.Hash == *uncle.Hash {
			c.logger.Error("uncle is a direct ancestor!!!")
			// uncle cannot be direct ancestor
			return false
		}
		for parent != nil && uncle != nil && *parent.Hash != *uncle.Hash {
			parent, _ = c.getChainNode(parent.Parent)
			uncle, _ = c.getChainNode(uncle.Parent)
			distance++
		}
		c.logger.Debug("Found uncle at distance: %d", distance)
//		return parent != nil && uncle != nil && (child.Depth().Uint64() - parent.Depth <  maxUncleDistance)
		return parent != nil && uncle != nil && (distance <=  maxUncleDistance)
	}
}

func (c *BlockChainConsensus) validateBlock(b Block) (*block, error) {
	_, ok := b.(*block)
	if !ok {
		c.logger.Error("attempt to submit incorrect block type: %T", b)
		return nil, core.NewCoreError(ERR_TYPE_INCORRECT, "block type incorrect")
	}
	// verify that block's parent exists
	var parent *block
	var err error
	if parent, err = c.getBlock(b.ParentHash()); err != nil {
		c.logger.Error("failed to find new block's parent: %s", err.Error())
		return nil, core.NewCoreError(ERR_BLOCK_ORPHAN, "cannot find parent")
	}
	// validate that block's depth is correct
	if b.Depth().Uint64() != parent.Depth().Uint64() + 1 {
		c.logger.Error("incorrect depth on new block")
		return nil, core.NewCoreError(ERR_BLOCK_VALIDATION, "incorrect depth")
	}
	// validate that block's weight is correct
	weight := parent.Weight().Uint64() + 1
	// validate uncles are known and within distance
	for _, uncle := range b.Uncles() {
		if !c.isUncleValid(b.(*block), &uncle) {
			c.logger.Error("block with invalid uncle: %x", uncle)
			return nil, core.NewCoreError(ERR_BLOCK_VALIDATION, "invalid uncle")
		}
		// increment weight for valid uncle
		weight++
	}

	if b.Weight().Uint64() != weight {
		c.logger.Error("incorrect weight on new block")
		return nil, core.NewCoreError(ERR_BLOCK_VALIDATION, "incorrect weight")
	}
	return parent, nil
}

// deserialize data into network block, and will initialize the block with block's parent's
// world state root (application is responsible to run the transactions from block, and update
// world state appropriately)
func (c *BlockChainConsensus) DeserializeNetworkBlock(data []byte) (Block, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
//	var block, parent *block
//	var err error
	if block, err := deSerializeBlock(data); err != nil {
		c.logger.Error("failed to deserialize network block's data: %s", err.Error())
		return nil, err
	} else {
		// process the block
		return c.processNetworkBlock(block)
	}
//	// set the network flag on block
//	block.isNetworkBlock = true
//
//	// validate block
//	if parent, err = c.validateBlock(block); err != nil {
//		return nil, err
//	}
//	// initialze block's world state to parent's world state
//	state := trie.NewMptWorldState(c.db)
//	if err = state.Rebase(parent.STATE); err != nil {
//		c.logger.Error("failed to initialize network block's world state: %s", err.Error())
//		return nil, core.NewCoreError(ERR_STATE_INCORRECT, "cannot initialize state")
//	}
//	block.worldState = state
//	return block, nil
}

func (c *BlockChainConsensus) DecodeNetworkBlock(msg p2p.Msg) (Block, error) {
//	c.lock.RLock()
//	defer c.lock.RUnlock()
	var spec BlockSpec
	if err := msg.Decode(&spec); err != nil {
		c.logger.Error("failed to decode p2p network block: %s", err)
		return nil, err
	}
	// process the block spec
	return c.DecodeNetworkBlockSpec(spec)
}

func (c *BlockChainConsensus) DecodeNetworkBlockSpec(spec BlockSpec) (Block, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	newBlock := &block{
		BlockSpec: BlockSpec {
			PHASH: spec.PHASH,
			MINER: spec.MINER,
			STATE: spec.STATE,
			TXs: make([]Transaction, len(spec.TXs)),
			TS: spec.TS,
			DEPTH: spec.DEPTH,
			WT: spec.WT,
			UNCLEs: make([]core.Byte64, len(spec.UNCLEs)),
			NONCE: spec.NONCE,
		},
	}
	for i,tx := range spec.TXs {
		newBlock.TXs[i] = tx
	}
	for i,uncle := range spec.UNCLEs {
		newBlock.UNCLEs[i] = uncle
	}
	// process the block
	return c.processNetworkBlock(newBlock)
}

func (c *BlockChainConsensus) processNetworkBlock(b *block) (Block, error) {
	// set the network flag on block
	b.isNetworkBlock = true
	b.computeHash()

	// validate block
	var parent *block
	var err error
	if parent, err = c.validateBlock(b); err != nil {
		return nil, err
	}
	// initialze block's world state to parent's world state
	state := trie.NewMptWorldState(c.db)
	if err = state.Rebase(parent.STATE); err != nil {
		c.logger.Error("failed to initialize network block's world state: %s", err.Error())
		return nil, core.NewCoreError(ERR_STATE_INCORRECT, "cannot initialize state")
	}
	b.worldState = state
	return b, nil
}

// submit a "processed" network block, will be added to DAG appropriately
// (i.e. either extend canonical chain, or add as an uncle block)
// block's computed world state should match STATE of the deSerialized block,
func (c *BlockChainConsensus) AcceptNetworkBlock(b Block) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// validate its a valid block
	var parent *block
	var err error
	if parent, err = c.validateBlock(b); err != nil {
		return err
	}
	// validate that computed state by application matches deserialized state of the block
	if b.(*block).worldState.Hash() != b.(*block).STATE {
		c.logger.Error("computed world state of network block incorrect")
		return core.NewCoreError(ERR_STATE_INCORRECT, "incorrect computed state")
	}
	return c.addValidatedBlock(b.(*block), parent)
}

// block has been validated (either mined local block, or processed network block) 
func (c *BlockChainConsensus) addValidatedBlock(child, parent *block) error {
	// verify that this is not a duplicate block
	var err error
	if _, err = c.getChainNode(child.Hash()); err == nil {
		c.logger.Error("block already exists: %x", *child.Hash())
		return core.NewCoreError(ERR_DUPLICATE_BLOCK, "duplicate block")
	}
	// add the new child node into our data store
	childNode := newChainNode(child)
	c.putBlock(child)
	// fetch parent's chain node
	var parentNode *chainNode
	if parentNode,err = c.getChainNode(parent.Hash()); err != nil {
		c.logger.Error("failed to find chain node for block!!!: %s", err)
		return core.NewCoreError(ERR_DB_CORRUPTED, "missing chain node")
	}
	// update parent's children list
	parentNode.addChild(child.Hash())
	c.putChainNode(parentNode)
	c.logger.Debug("adding a new block at depth '%d' in the block chain", childNode.depth())
	// compare current main list weight with weight of new node's list
	// to find if main list needs rebalancing
	if c.weight < childNode.weight() {
		c.logger.Debug("rebalancing the block chain after new block addition")
		// move depth and tip of blockchain
		c.weight = childNode.weight()
		c.tip = child
		// update the tip in DB
		if err := c.db.Put(dagTip, c.tip.Hash().Bytes()); err != nil {
			return core.NewCoreError(ERR_DB_CORRUPTED, "failed to update tip in DB")
		}
		// change the world state
		if err := c.state.Rebase(child.STATE); err != nil {
			return core.NewCoreError(ERR_DB_CORRUPTED, "failed to update world state")
		}
		
		// walk up the ancestor list setting them up as main list nodes
		// until find the first ancestor that is already on main list
		childNode.setMainList(true)
		mainListParent := childNode
		for !parentNode.isMainList() {
			parentNode.setMainList(true)
			c.putChainNode(parentNode)
			mainListParent = parentNode
			parentNode, _ = c.getChainNode(parentNode.parent())
		}
		// find the original main list child
		c.putChainNode(childNode)
		childNode = c.findMainListChild(parentNode, mainListParent)
		// walk down the old main list and reset flag
		for childNode != nil {
			c.logger.Debug("removing block at depth '%d' from old main list", childNode.depth())
			childNode.setMainList(false)
			c.putChainNode(childNode)
			childNode = c.findMainListChild(childNode, nil)
		}
	} else {
		c.putChainNode(childNode)
		c.logger.Debug("block is not on mainlist")
	}
	return nil
}

// TODO: optimize this by adding mainlist flag in the children itself, so that dont have to make 2nd DB dip just to find that
func (c *BlockChainConsensus) findMainListChild(parent, skipChild *chainNode) *chainNode {
	for _, childHash := range (parent.children()) {
		child, _ := c.getChainNode(childHash)
		if child != nil && child.isMainList() && (skipChild == nil || *skipChild.hash() != *child.hash()) {
			return child
		}
	}
	return nil
}

func (c *BlockChainConsensus) Block(hash *core.Byte64) (Block, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if block, err := c.getBlock(hash); err != nil {
		return nil, err
	} else {
		// create a copy of world state from block's world view
		state := trie.NewMptWorldState(c.db)
		if err := state.Rebase(block.STATE); err != nil {
			c.logger.Error("failed to initialize block's world state: %s", err.Error())
			return nil, core.NewCoreError(ERR_STATE_INCORRECT, "worldstate error")
		}
		return block, nil
	}
}

// a copy of best block in current cannonical chain, used by protocol manager for handshake
func (c *BlockChainConsensus) BestBlock() Block {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// create a copy of world state from current canonical tip's world view
	state := trie.NewMptWorldState(c.db)
	if err := state.Rebase(c.state.Hash()); err != nil {
		c.logger.Error("failed to initialize best block's world state: %s", err.Error())
		return nil
	}
	return c.tip.clone(state)
}

// ordered list of serialized descendents from specific parent, on the current canonical chain
//func (c *BlockChainConsensus) Descendents(parent *core.Byte64, max int) ([][]byte, error) {
func (c *BlockChainConsensus) Descendents(parent *core.Byte64, max int) ([]Block, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// limit number of blocks to maxBlocks
	if max > maxBlocks {
		max = maxBlocks
	}
//	// create placeholder for deserialized descendents
//	descendents := make([][]byte, 0, max)
	// create placeholder for  descendents
	descendents := make([]Block, 0, max)

	var parentNode *chainNode
	var err error
	if parentNode, err = c.getChainNode(parent); err != nil {
		c.logger.Error("failed to fetch parent of descendents: %s", err.Error())
		return descendents, err
	}
	// validate that its on canonical chain
	if !parentNode.isMainList() {
		c.logger.Error("fetched parent of descendent not on mainlist")
		return descendents, core.NewCoreError(ERR_NOT_MAINLIST, "parent not on mainlist")
	}
	// loop over fetching chainNode and its mainlist descendents
	count := 0
	for childNode := c.findMainListChild(parentNode, nil); childNode != nil && count < max; count++{
		// fetch actual block for this child node
		if child, err := c.getBlock(childNode.hash()); err != nil {
			c.logger.Error("failed to fetch descendent block: %s", err.Error())
			return descendents, err
		} else {
			// first fetch all uncles of this block
			if (count + len(child.Uncles())) >= max {
					c.logger.Debug("skipping block since uncles also need to be included in next batch")
					break
			}
			uncles := make([]Block, 0, len(child.Uncles()))
			for _, hash := range child.Uncles() {
				if uncle, err := c.getBlock(&hash); err != nil {
					c.logger.Error("failed to fetch uncle block: %s", err.Error())
					return descendents, err
				} else {
					c.logger.Debug("adding uncle block: %x", *uncle.Hash())
					uncles = append(uncles, uncle)
					count++
				}
			}
			descendents = append(descendents, uncles...)
			c.logger.Debug("adding descendent block: %x", *child.Hash())
			descendents = append(descendents, child)
		}
		// move down descendent list
		childNode = c.findMainListChild(childNode, nil)
	}
	
	return descendents, nil
}