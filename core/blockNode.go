package core

import (
	"sync"
)

type BlockNode struct {
	hash *Byte64
	parent *Byte64
	depth uint64
	isMainList bool
	children	 []*Byte64
	block Block
	lock sync.RWMutex
}

func NewBlockNode(block Block, depth uint64) *BlockNode {
	return &BlockNode{
		hash: block.Hash(),
		parent: block.Previous().Bytes(),
		block: block,
		depth: depth,
		isMainList: false,
		children: make([]*Byte64,0),
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

func (bn *BlockNode) Block() Block {
	return bn.block
}

func (bn *BlockNode) Hash() *Byte64 {
	return bn.hash
}

func (bn *BlockNode) Parent() *Byte64 {
	return bn.parent
}

func (bn *BlockNode) Depth() uint64 {
	return bn.depth
}

func (bn *BlockNode) Children() []*Byte64 {
	return bn.children
}

func (bn *BlockNode) AddChild(childHash *Byte64) int {
	bn.children = append(bn.children, childHash)
	return len(bn.children)
}
