package core

import (
	"sync"
	"github.com/trust-net/go-trust-net/log"

)

type DAG interface {
	
}

const (
	maxBlocks = 100
)

type BlockChainInMem struct {
	genesis *BlockNode
	leaves []*BlockNode // do we need this?
	td	*Byte8
	depth uint64
	nodes map[Byte64]*BlockNode
	lock sync.RWMutex
	logger log.Logger
}

// we should pass genesis block as parameter, instead of auto generating here
// since genesis block needs to be the "same" on all nodes/instances of this blockchain
//
// this way, an exactly same genesis block can be computed deterministically from a config
// on all instances of this blockchain
func NewBlockChainInMem(genesis Block) *BlockChainInMem {
	chain := &BlockChainInMem{
		genesis: NewBlockNode(genesis, 0),
		depth: 0,
		td: BytesToByte8(genesis.Timestamp().Bytes()),
		leaves: make([]*BlockNode, 1), // do we need this?
		nodes: make(map[Byte64]*BlockNode),
	}
	chain.nodes[*genesis.Hash()] = chain.genesis
	chain.genesis.SetMainList(true)
	chain.leaves[0]=chain.genesis
	chain.logger = log.NewLogger(*chain)
	chain.logger.Debug("Created new instance of in memory block chain DAG")
	return chain
}

func (chain *BlockChainInMem) Depth() uint64 {
	return chain.depth
}

func (chain *BlockChainInMem) TD() *Byte8 {
	return chain.td
}

func (chain *BlockChainInMem) Tip() *BlockNode {
	return chain.leaves[0]
}

func (chain *BlockChainInMem) Genesis() *BlockNode {
	return chain.genesis
}

func (chain *BlockChainInMem) BlockNode(hash *Byte64) (*BlockNode, bool) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	node, found := chain.nodes[*hash]
	return node, found
}

func (chain *BlockChainInMem) AddBlockNode(block Block) error {
	if block == nil {
		chain.logger.Error("attempt to add nil block!!!")
		return NewCoreError(ERR_INVALID_BLOCK, "nil block")
	}
	// make sure that block has computed hash
	if block.Hash() == nil {
		chain.logger.Error("attempt to add block without hash computed")
		return NewCoreError(ERR_INVALID_HASH, "block does not have hash computed")
	}
	chain.lock.Lock()
	defer chain.lock.Unlock()
	if _, found := chain.nodes[*block.Hash()]; found {
		chain.logger.Error("attempt to add duplicate block!!!")
		return NewCoreError(ERR_DUPLICATE_BLOCK, "duplicate block")
	}
	if parent, ok := chain.nodes[*block.ParentHash()]; !ok {
		chain.logger.Error("attempt to add an orphan block!!!")
		return NewCoreError(ERR_ORPHAN_BLOCK, "orphan block")
	} else {
		// add the new child node into our data store
		child := NewBlockNode(block, parent.Depth()+1)
		chain.nodes[*child.Hash()] = child
		// update parent's children list
		parent.AddChild(child.Hash())
		chain.logger.Debug("adding a new block at depth '%d' in the block chain", child.Depth())
		// compare current main list depth with depth of new node's list
		// to find if main list needs rebalancing
		if chain.Depth() < child.Depth() {
			chain.logger.Debug("rebalancing the block chain after new block addition")
			// move depth and tip of blockchain
			*chain.td = *block.Timestamp()
			chain.depth = child.Depth()
			chain.leaves[0] = child
			// walk up the ancestor list setting them up as main list nodes
			// until find the first ancestor that is already on main list
			child.SetMainList(true)
			mainListParent := child
			for !parent.IsMainList() {
				parent.SetMainList(true)
				mainListParent = parent
				parent = chain.nodes[*parent.Parent()]
			}
			// find the original main list child
			child = chain.findMainListChild(parent, mainListParent)
			// walk down the old main list and reset flag
			for child != nil {
				chain.logger.Debug("removing block at depth '%d' from old main list", child.Depth())
				child.SetMainList(false)
				child = chain.findMainListChild(child, nil)
			}
		}
	}
	return nil
}

func (chain *BlockChainInMem) findMainListChild(parent, skipChild *BlockNode) *BlockNode {
	for _, childHash := range (parent.Children()) {
		child := chain.nodes[*childHash]
		if child.IsMainList() && (skipChild == nil || skipChild.Hash() != child.Hash()) {
			return child
		}
	}
	return nil
}

func (chain *BlockChainInMem) Blocks(parent *Byte64, max uint64) []Block {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	if max > maxBlocks {
		max = maxBlocks
	}
	blocks := make([]Block, 0, max)
	// simple traversal down the main block chain list
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	// skip the parent
	currNode, count := chain.nodes[*parent], uint64(0)
	if currNode != nil {
		currNode = chain.findMainListChild(currNode, nil)
	}
	for ; currNode != nil && count < max; {
		chain.logger.Debug("Traversing block at depth '%d' on main list", currNode.Depth())
		blocks = append(blocks, currNode.Block())
		currNode = chain.findMainListChild(currNode, nil)
		count++
	}
	return blocks
}
