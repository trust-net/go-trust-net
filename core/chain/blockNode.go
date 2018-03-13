package chain

import (
//	"sync"
	"github.com/trust-net/go-trust-net/core"
)

type BlockNode struct {
	HASH *core.Byte64
	PARENT *core.Byte64
	DEPTH uint64
	WEIGHT uint64
	ISMAINLIST bool
	CHILDREN	 []*core.Byte64
	BLOCK *core.Byte64
}

func NewBlockNode(block core.Block) *BlockNode {
	return &BlockNode{
		HASH: block.Hash(),
		PARENT: block.ParentHash(),
		BLOCK: block.Hash(),
		WEIGHT: block.Weight().Uint64(),
		DEPTH: block.Depth().Uint64(),
		ISMAINLIST: false,
		CHILDREN: make([]*core.Byte64,0),
	}
}

func (bn *BlockNode) IsMainList() bool{
	return bn.ISMAINLIST
}

func (bn *BlockNode) SetMainList(isMainList bool) {
	bn.ISMAINLIST = isMainList
}

func (bn *BlockNode) Block() *core.Byte64 {
	return bn.BLOCK
}

func (bn *BlockNode) Hash() *core.Byte64 {
	return bn.HASH
}

func (bn *BlockNode) Parent() *core.Byte64 {
	return bn.PARENT
}

func (bn *BlockNode) Depth() uint64 {
	return bn.DEPTH
}

func (bn *BlockNode) Weight() uint64 {
	return bn.WEIGHT
}

func (bn *BlockNode) Children() []*core.Byte64 {
	return bn.CHILDREN
}

func (bn *BlockNode) AddChild(childHash *core.Byte64) int {
	bn.CHILDREN = append(bn.CHILDREN, childHash)
	return len(bn.CHILDREN)
}
