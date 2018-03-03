package chain

import (
	"sync"
	"github.com/trust-net/go-trust-net/core"
)

type BlockNode struct {
	hash *core.Byte64
	parent *core.Byte64
	depth uint64
	isMainList bool
	children	 []*core.Byte64
	block core.Block
	lock sync.RWMutex
}

func NewBlockNode(block core.Block, depth uint64) *BlockNode {
	return &BlockNode{
		hash: block.Hash(),
		parent: block.ParentHash(),
		block: block,
		depth: depth,
		isMainList: false,
		children: make([]*core.Byte64,0),
	}
}

func (bn *BlockNode) IsMainList() bool{
	return bn.isMainList
}

func (bn *BlockNode) SetMainList(isMainList bool) {
	bn.isMainList = isMainList
}

func (bn *BlockNode) Lock() {
	bn.lock.Lock()
}

func (bn *BlockNode) Unlock() {
	bn.lock.Unlock()
}

func (bn *BlockNode) Block() core.Block {
	return bn.block
}

func (bn *BlockNode) Hash() *core.Byte64 {
	return bn.hash
}

func (bn *BlockNode) Parent() *core.Byte64 {
	return bn.parent
}

func (bn *BlockNode) Depth() uint64 {
	return bn.depth
}

func (bn *BlockNode) Children() []*core.Byte64 {
	return bn.children
}

func (bn *BlockNode) AddChild(childHash *core.Byte64) int {
	bn.children = append(bn.children, childHash)
	return len(bn.children)
}
