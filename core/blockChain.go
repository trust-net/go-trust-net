package core

import (
	"time"
	"sync"

)

type DAG interface {
	
}

const (
	maxBlocks = 100
)

type BlockChainInMem struct {
	genesis *BlockNode
	leaves []*BlockNode // do we need this?
	td	time.Duration
	depth uint64
	nodes map[Byte64]*BlockNode
	lock sync.RWMutex
}

func NewBlockChainInMem() *BlockChainInMem {
	genesis := NewSimpleBlock(BytesToByte64(nil), BytesToByte64(nil), NewSimpleNodeInfo(""))
	genesis.ComputeHash()
	chain := &BlockChainInMem{
		genesis: NewBlockNode(genesis, 0),
		depth: 0,
		td: genesis.TD(),
		leaves: make([]*BlockNode, 1), // do we need this?
		nodes: make(map[Byte64]*BlockNode),
	}
	chain.nodes[*genesis.Hash()] = chain.genesis
	return chain
}

func (chain *BlockChainInMem) Depth() uint64 {
	return chain.depth
}

func (chain *BlockChainInMem) TD() time.Duration {
	return chain.td
}

func (chain *BlockChainInMem) BlockNode(hash *Byte64) (*BlockNode, bool) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	node, found := chain.nodes[*hash]
	return node, found
}

func (chain *BlockChainInMem) AddBlockNode(block Block) error {
	if block == nil {
		return NewCoreError("nil block")
	}
	// make sure that block has computed hash
	if block.Hash() == nil {
		return NewCoreError("block does not have hash computed")
	}
	chain.lock.Lock()
	defer chain.lock.Unlock()
	if _, found := chain.nodes[*block.Hash()]; found {
		return NewCoreError("duplicate block")
	}
	if parent, ok := chain.nodes[*block.Previous().Bytes()]; !ok {
		return NewCoreError("orphan block")
	} else {
		// add the node into our data store
		node := NewBlockNode(block, parent.Depth()+1)
		chain.nodes[*node.Hash()] = node
		// update parent's children list
		parent.AddChild(node.Hash())
		// update longest chain marker
		if chain.depth < node.Depth() {
			chain.td = block.TD()
			chain.depth = node.Depth()
			chain.leaves[0] = node // do we need this?
		}
	}
	return nil
}


func (chain *BlockChainInMem) Blocks(parent *Byte64, max uint64) []*BlockNode {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	if max > maxBlocks {
		max = maxBlocks
	}
	blocks := make([]*BlockNode, 0, max)
	// TODO traverse DAG and fill in the list
	return blocks
}
