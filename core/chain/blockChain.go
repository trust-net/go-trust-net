package chain

import (
	"sync"
	"encoding/gob"
    "bytes"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"

)

const (
	maxBlocks = 100
)

var tableBlockNode = []byte("BlockNode-")
var tableBlockSpec = []byte("BlockSpec-")

func init() {
	gob.Register(&BlockNode{})
	gob.Register(&core.BlockSpec{})
}

func tableKey(prefix []byte, key *core.Byte64) []byte {
	return append(prefix, key.Bytes()...)
}

type BlockChainInMem struct {
	genesis *BlockNode
	tip *BlockNode
	td	*core.Byte8
	depth uint64
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
		genesis: NewBlockNode(genesis, 0),
		depth: 0,
		td: core.BytesToByte8(genesis.Timestamp().Bytes()),
		db: db,
	}
	chain.tip = chain.genesis
	chain.logger = log.NewLogger(*chain)
	chain.logger.Debug("Created new instance of in memory block chain DAG")
	chain.genesis.SetMainList(true)
	if err := chain.SaveBlock(genesis); err != nil {
		return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
	}
	if err := chain.SaveBlockNode(chain.genesis); err != nil {
		return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
	}
	return chain, nil
}

func (chain *BlockChainInMem) Shutdown() error {
	return chain.db.Close()
}

func (chain *BlockChainInMem) Depth() uint64 {
	return chain.depth
}

func (chain *BlockChainInMem) TD() *core.Byte8 {
	return chain.td
}

func (chain *BlockChainInMem) Tip() *BlockNode {
	return chain.tip
}

func (chain *BlockChainInMem) Genesis() *BlockNode {
	return chain.genesis
}

func encode(entity interface{}) ([]byte, error) {
	b := bytes.Buffer{}
    e := gob.NewEncoder(&b)
    if err := e.Encode(entity); err != nil {
	    	return []byte{}, err
    } else {
	    	return b.Bytes(), nil
    }
}

func decode(data []byte, entity interface{}) error {
	b := bytes.Buffer{}
    b.Write(data)
    d := gob.NewDecoder(&b)
    return d.Decode(entity)
}

func (chain *BlockChainInMem) BlockNode(hash *core.Byte64) (*BlockNode, bool) {
	if data, err := chain.db.Get(tableKey(tableBlockNode, hash)); err != nil {
		chain.logger.Error("Did not find block node in DB: %s", err.Error())
		return nil, false
	} else {
		var node BlockNode
		if err := decode(data, &node); err != nil {
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
		if err := decode(data, &spec); err != nil {
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
		if err := decode(data, &spec); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, false
		}
		return &spec, true
	}
}

// persist a blocknode into DB
func (chain *BlockChainInMem) SaveBlockNode(node *BlockNode) error {
	if data, err := encode(node); err == nil {
		return chain.db.Put(tableKey(tableBlockNode, node.Hash()), data)
	} else {
		return err
	}
	
}

// persist a block into DB
func (chain *BlockChainInMem) SaveBlock(block core.Block) error {
	if data, err := encode(core.NewBlockSpecFromBlock(block)); err == nil {
		return chain.db.Put(tableKey(tableBlockSpec, block.Hash()), data)
	} else {
		return err
	}
}

func (chain *BlockChainInMem) AddBlockNode(block core.Block) error {
	if block == nil {
		chain.logger.Error("attempt to add nil block!!!")
		return core.NewCoreError(core.ERR_INVALID_BLOCK, "nil block")
	}
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
		// add the new child node into our data store
		child := NewBlockNode(block, parent.Depth()+1)
		chain.SaveBlock(block)
		// update parent's children list
		parent.AddChild(child.Hash())
		chain.SaveBlockNode(parent)
		chain.logger.Debug("adding a new block at depth '%d' in the block chain", child.Depth())
		// compare current main list depth with depth of new node's list
		// to find if main list needs rebalancing
		if chain.Depth() < child.Depth() {
			chain.logger.Debug("rebalancing the block chain after new block addition")
			// move depth and tip of blockchain
			*chain.td = *block.Timestamp()
			chain.depth = child.Depth()
			chain.tip = child
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
	}
	return nil
}

// TODO: optimize this by adding mainlist flag in the children itself, so that dont have to make 2nd DB dip just to find that
func (chain *BlockChainInMem) findMainListChild(parent, skipChild *BlockNode) *BlockNode {
	for _, childHash := range (parent.Children()) {
		child, _ := chain.BlockNode(childHash)
		if child.IsMainList() && (skipChild == nil || *skipChild.Hash() != *child.Hash()) {
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
