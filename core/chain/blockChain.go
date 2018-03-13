package chain

import (
	"sync"
	"encoding/gob"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/db"

)

const (
	maxBlocks = 100
	maxUncleDistance = uint64(5)
)

var tableBlockNode = []byte("BlockNode-")
var tableBlockSpec = []byte("BlockSpec-")
var dagTip = []byte("ChainState-DagTip")

func init() {
	gob.Register(&BlockNode{})
	gob.Register(&core.BlockSpec{})
}

func tableKey(prefix []byte, key *core.Byte64) []byte {
	return append(prefix, key.Bytes()...)
}

type BlockChainInMem struct {
	genesis *BlockNode
	tip *core.Byte64
	td	*core.Byte8
	depth uint64
	weight uint64
	db db.Database
	lock sync.RWMutex
	logger log.Logger
}

// we should pass genesis block as parameter, instead of auto generating here
// since genesis block needs to be the "same" on all nodes/instances of this blockchain
//
// this way, an exactly same genesis block can be computed deterministically from a config
// on all instances of this blockchain
func NewBlockChainInMem(genesis core.Block, db db.Database) (*BlockChainInMem, error) {
	chain := &BlockChainInMem{
		genesis: NewBlockNode(genesis),
		depth: 0,
		td: core.BytesToByte8(genesis.Timestamp().Bytes()),
		db: db,
	}
	chain.genesis.SetMainList(true)
	chain.logger = log.NewLogger(*chain)
	// initialize tip from DB, if available
	if data, err := chain.db.Get(dagTip); err != nil {
		chain.logger.Debug("No DAG tip in DB, using genesis as tip")
		chain.tip = chain.genesis.Hash()
		// save the tip in DB
		if err := chain.db.Put(dagTip, chain.tip.Bytes()); err != nil {
			return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
		}
		// save the genesis block in DB
		if err := chain.SaveBlock(genesis); err != nil {
			return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
		}
		if err := chain.SaveBlockNode(chain.genesis); err != nil {
			return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
		}
	} else {
		chain.tip = core.BytesToByte64(data)
		chain.logger.Debug("Initializing DAG tip from DB: '%x'", *chain.tip)
		// find depth from tip
		if node, found := chain.BlockNode(chain.tip); !found {
			return nil, core.NewCoreError(core.ERR_DB_CORRUPTED, "cannot find block node for the tip")
		} else {
			chain.depth = node.Depth()
		}
		// find TD from tip
		if block, found := chain.Block(chain.tip); !found {
			return nil, core.NewCoreError(core.ERR_DB_CORRUPTED, "cannot find block for the tip")
		} else {
			chain.td = core.BytesToByte8(block.Timestamp().Bytes())
		}
	}
	chain.logger.Debug("Initialized block chain DAG")
	return chain, nil
}

func (chain *BlockChainInMem) Shutdown() error {
	return chain.db.Close()
}

// reset chain DB and flush all existing data
func (chain *BlockChainInMem) Flush() error {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	chain.logger.Debug("Flushing DB and resetting chain DAG")
	// reset tip to genesis block
	chain.tip = chain.genesis.Hash()
	// save the tip in DB
	if err := chain.db.Put(dagTip, chain.tip.Bytes()); err != nil {
		return core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
	}
	chain.depth = 0
	// read the genesis from DB, to also retreive descendents
	genesisNode, _ := chain.BlockNode(chain.genesis.Hash()) 
	genesis, _ := chain.Block(chain.genesis.Hash())
	chain.td = genesis.Timestamp()
	// walk down the mainlist and delete blocks
	i := 1
	for child := chain.findMainListChild(genesisNode, nil); child != nil; {
		chain.logger.Debug("[%05d] : deleting '%x'", i, child.Hash())
		i++
		if err := chain.db.Delete(tableKey(tableBlockNode, child.Hash())); err != nil {
			chain.logger.Error("failed to remove block node from DAG: %s", err.Error())
		}
		if err := chain.db.Delete(tableKey(tableBlockSpec, child.Hash())); err != nil {
			chain.logger.Error("failed to remove block from DB: %s", err.Error())
		}
		child = chain.findMainListChild(child, nil)
	}
	return nil
}

func (chain *BlockChainInMem) Depth() uint64 {
	return chain.depth
}

func (chain *BlockChainInMem) Weight() uint64 {
	return chain.weight
}

func (chain *BlockChainInMem) TD() *core.Byte8 {
	return chain.td
}

func (chain *BlockChainInMem) Tip() *BlockNode {
	// get the blocknode for the tip
	if data, err := chain.db.Get(tableKey(tableBlockNode, chain.tip)); err != nil {
		chain.logger.Error("Did not find tip block node in DB: %s", err.Error())
		return nil
	} else {
		var node BlockNode
		if err := common.Deserialize(data, &node); err != nil {
			chain.logger.Error("failed to decode tip data from DB: %s", err.Error())
			return nil
		}
		return &node
	}
}

func (chain *BlockChainInMem) Genesis() *BlockNode {
	return chain.genesis
}

func (chain *BlockChainInMem) BlockNode(hash *core.Byte64) (*BlockNode, bool) {
	if data, err := chain.db.Get(tableKey(tableBlockNode, hash)); err != nil {
		chain.logger.Error("Did not find block node in DB: %s", err.Error())
		return nil, false
	} else {
		var node BlockNode
		if err := common.Deserialize(data, &node); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, false
		}
		return &node, true
	}
}

func (chain *BlockChainInMem) Block(hash *core.Byte64) (core.Block, bool) {
	if data, err := chain.db.Get(tableKey(tableBlockSpec, hash)); err != nil {
		chain.logger.Error("Did not find block in DB: %s", err.Error())
		return nil, false
	} else {
		var spec core.BlockSpec
		if err := common.Deserialize(data, &spec); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, false
		}
		return core.NewSimpleBlockFromSpec(&spec), true
	}
}

func (chain *BlockChainInMem) BlockSpec(hash *core.Byte64) (*core.BlockSpec, bool) {
	if data, err := chain.db.Get(tableKey(tableBlockSpec, hash)); err != nil {
		chain.logger.Error("Did not find block in DB: %s", err.Error())
		return nil, false
	} else {
		var spec core.BlockSpec
		if err := common.Deserialize(data, &spec); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, false
		}
		return &spec, true
	}
}

// persist a blocknode into DB
func (chain *BlockChainInMem) SaveBlockNode(node *BlockNode) error {
	if data, err := common.Serialize(node); err == nil {
		return chain.db.Put(tableKey(tableBlockNode, node.Hash()), data)
	} else {
		return err
	}
	
}

// persist a block into DB
func (chain *BlockChainInMem) SaveBlock(block core.Block) error {
	if data, err := common.Serialize(core.NewBlockSpecFromBlock(block)); err == nil {
		return chain.db.Put(tableKey(tableBlockSpec, block.Hash()), data)
	} else {
		return err
	}
}

type uncle struct {
	hash *core.Byte64
	miner *core.Byte64
	depth uint64
	distance uint64
}

func (chain *BlockChainInMem) findNonDirectAncestors(childNode *core.Byte64, remainingDistance, maxDepth uint64) []uncle {
	uncles := make([]uncle, 0, 5)
	if remainingDistance == 0 {
		chain.logger.Debug("reached max uncle search: remainingDistance %d", remainingDistance)
		return uncles
	}
	
	if node, found := chain.BlockNode(childNode); found {
		for _, grandChild := range node.Children() {
			if grandChild != nil {
				if grandChildBlock, found := chain.Block(grandChild); found && grandChildBlock.Depth().Uint64() <= maxDepth {
					chain.logger.Debug("Found uncle: %x, remainingDistance: %d, depth %d", *grandChildBlock.Hash(), remainingDistance, grandChildBlock.Depth().Uint64())
					uncles = append(uncles, uncle{
							hash: grandChildBlock.Hash(),
							miner: grandChildBlock.Miner(),
							depth: grandChildBlock.Depth().Uint64(),
							distance: maxUncleDistance - remainingDistance,
							
					})
					uncles = append(uncles, chain.findNonDirectAncestors(grandChild, remainingDistance-1, maxDepth)...)
				} 
			}
		}
	}
	return uncles
}

func (chain *BlockChainInMem) findUncles(grandParent, parent *core.Byte64, remainingDistance, maxDepth uint64) []uncle {
	uncles := make([]uncle, 0, 5)
	if remainingDistance == 0 {
//		chain.logger.Debug("reached max uncle search: remainingDistance %d", remainingDistance)
		return uncles
	}
	if node, found := chain.BlockNode(grandParent); found {
		for _, childNode := range node.Children() {
			if childNode != nil && *childNode != *parent {
				if childBlock, found := chain.Block(childNode); found {
					chain.logger.Debug("Found uncle: %x, remainingDistance: %d, depth %d", *childBlock.Hash(), remainingDistance, childBlock.Depth().Uint64())
					uncles = append(uncles, uncle{
							hash: childBlock.Hash(),
							miner: childBlock.Miner(),
							depth: childBlock.Depth().Uint64(),
							distance: maxUncleDistance - remainingDistance + 1,
							
					})
					uncles = append(uncles, chain.findNonDirectAncestors(childNode, remainingDistance-1, maxDepth)...)
				} 
			}
		}
		uncles = append(uncles, chain.findUncles(node.Parent(), node.Hash(), remainingDistance-1, maxDepth)...)
	}
	return uncles
}

func (chain *BlockChainInMem) addBlock(block *core.SimpleBlock, parent *BlockNode) error {
	// add the new child node into our data store
	child := NewBlockNode(block)
	chain.SaveBlock(block)
	// update parent's children list
	parent.AddChild(child.Hash())
	chain.SaveBlockNode(parent)
	chain.logger.Debug("adding a new block at depth '%d' in the block chain", child.Depth())
	// compare current main list weight with weight of new node's list
	// to find if main list needs rebalancing
	if chain.Weight() < child.Weight() {
		chain.logger.Debug("rebalancing the block chain after new block addition")
		// move depth and tip of blockchain
		*chain.td = *block.Timestamp()
		chain.depth = child.Depth()
		chain.weight = child.Weight()
		chain.tip = child.Hash()
		// update the tip in DB
		if err := chain.db.Put(dagTip, chain.tip.Bytes()); err != nil {
			return core.NewCoreError(core.ERR_DB_CORRUPTED, "failed to update tip in DB")
		}
		
		// walk up the ancestor list setting them up as main list nodes
		// until find the first ancestor that is already on main list
		child.SetMainList(true)
		mainListParent := child
		for !parent.IsMainList() {
			parent.SetMainList(true)
			chain.SaveBlockNode(parent)
			mainListParent = parent
			parent, _ = chain.BlockNode(parent.Parent())
		}
		// find the original main list child
		chain.SaveBlockNode(child)
		child = chain.findMainListChild(parent, mainListParent)
		// walk down the old main list and reset flag
		for child != nil {
			chain.logger.Debug("removing block at depth '%d' from old main list", child.Depth())
			child.SetMainList(false)
			chain.SaveBlockNode(child)
			child = chain.findMainListChild(child, nil)
		}
	} else {
		chain.SaveBlockNode(child)
		chain.logger.Debug("block is not on mainlist")
	}
	return nil
}

func (chain *BlockChainInMem) AddBlockNode(coreBlock core.Block) error {
	if coreBlock == nil {
		chain.logger.Error("attempt to add nil block!!!")
		return core.NewCoreError(core.ERR_INVALID_BLOCK, "nil block")
	}
	block := coreBlock.(*core.SimpleBlock)
	// make sure that block has nil hash
	if block.Hash() != nil {
		chain.logger.Error("attempt to add new block with precomputed hash")
		return core.NewCoreError(core.ERR_INVALID_BLOCK, "block has precomputed hash")
	}
	chain.lock.Lock()
	defer chain.lock.Unlock()
	if parent, found := chain.BlockNode(block.ParentHash()); !found {
		chain.logger.Error("attempt to add an orphan block!!!")
		return core.NewCoreError(core.ERR_ORPHAN_BLOCK, "orphan block")
	} else {
		if len(block.Uncles()) != 0 {
			chain.logger.Error("new block already has %d uncles!!!", len(block.Uncles()))
			return core.NewCoreError(core.ERR_INVALID_BLOCK, "new block cannot have uncles")
		}
		// find uncles, if available, and adjust weight
		for _, uncle := range chain.findUncles(parent.Parent(), parent.Hash(), maxUncleDistance, parent.Depth()) {
			chain.logger.Debug("Adding %d distant uncle: %x, miner: %x", uncle.distance, *uncle.hash, *uncle.miner)
			block.AddUncle(uncle.hash, 1)
			// TODO: process mining reward for uncle list
		}
		block.ComputeHash()
		if found, _ := chain.db.Has(tableKey(tableBlockNode, block.Hash())); found {
			chain.logger.Error("attempt to add duplicate block!!!")
			return core.NewCoreError(core.ERR_DUPLICATE_BLOCK, "duplicate block")
		}
		return chain.addBlock(block, parent)
	}
}


func (chain *BlockChainInMem) AddNetworkNode(coreBlock core.Block) error {
	if coreBlock == nil {
		chain.logger.Error("attempt to add nil block!!!")
		return core.NewCoreError(core.ERR_INVALID_BLOCK, "nil block")
	}
	block := coreBlock.(*core.SimpleBlock)
	// make sure that block has computed hash
	if block.Hash() == nil {
		chain.logger.Error("attempt to add block without hash computed")
		return core.NewCoreError(core.ERR_INVALID_HASH, "block does not have hash computed")
	}
	chain.lock.Lock()
	defer chain.lock.Unlock()
	if found, _ := chain.db.Has(tableKey(tableBlockNode, block.Hash())); found {
		chain.logger.Error("attempt to add duplicate block!!!")
		return core.NewCoreError(core.ERR_DUPLICATE_BLOCK, "duplicate block")
	}
	if parent, found := chain.BlockNode(block.ParentHash()); !found {
		chain.logger.Error("attempt to add an orphan block!!!")
		return core.NewCoreError(core.ERR_ORPHAN_BLOCK, "orphan block")
	} else {
		// validate uncles are known and within distance
		for _, uncle := range block.Uncles() {
			if found, _ := chain.db.Has(tableKey(tableBlockNode, uncle)); !found {
				chain.logger.Error("attempt to add block with unknown uncle!!!")
				return core.NewCoreError(core.ERR_INVALID_UNCLE, "unknown uncle")
			}
			// TODO: process mining reward for uncle list
		}
		return chain.addBlock(block, parent)
	}
}

// TODO: optimize this by adding mainlist flag in the children itself, so that dont have to make 2nd DB dip just to find that
func (chain *BlockChainInMem) findMainListChild(parent, skipChild *BlockNode) *BlockNode {
	for _, childHash := range (parent.Children()) {
		child, _ := chain.BlockNode(childHash)
		if child != nil && child.IsMainList() && (skipChild == nil || *skipChild.Hash() != *child.Hash()) {
			return child
		}
	}
	return nil
}

func (chain *BlockChainInMem) Blocks(parent *core.Byte64, max uint64) []core.Block {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	if max > maxBlocks {
		max = maxBlocks
	}
	blocks := make([]core.Block, 0, max)
	// simple traversal down the main block chain list
	// skip the parent
	currNode, _ := chain.BlockNode(parent)
	if currNode != nil {
		currNode = chain.findMainListChild(currNode, nil)
	}
	for count := uint64(0); currNode != nil && count < max; {
		chain.logger.Debug("Traversing block at depth '%d' on main list", currNode.Depth())
		if block, found := chain.Block(currNode.Block()); found {
			blocks = append(blocks, block)
			currNode = chain.findMainListChild(currNode, nil)
			count++
		} else {
			break
		}
	}
	return blocks
}

func (chain *BlockChainInMem) BlockSpecs(parent *core.Byte64, max uint64) []core.BlockSpec {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	if max > maxBlocks {
		max = maxBlocks
	}
	blocks := make([]core.BlockSpec, 0, max)
	// simple traversal down the main block chain list
	// skip the parent
	currNode, _ := chain.BlockNode(parent)
	if currNode != nil {
		currNode = chain.findMainListChild(currNode, nil)
	}
	for count := uint64(0); currNode != nil && count < max; {
		chain.logger.Debug("Traversing block at depth '%d' on main list", currNode.Depth())
		if block, found := chain.BlockSpec(currNode.Block()); found {
			blocks = append(blocks, *block)
			currNode = chain.findMainListChild(currNode, nil)
			count++
		} else {
			break
		}
	}
	return blocks
}
