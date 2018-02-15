package core

import (
)

type BlockNode struct {
	hash []byte
	parent []byte
	depth uint64
	children	 [][]byte
}

//func NewBlockNode(hash, parent Byte32, depth uint64) *BlockNode {
//	return &BlockNode{
//		hash: hash,
//		parent: parent,
//		depth: depth,
//		children: make([]Byte32,0),
//	}
//}
//
func NewBlockNode(block Block, depth uint64) *BlockNode {
	return &BlockNode{
		hash: block.Hash(),
		parent: block.Previous().Bytes(),
		depth: depth,
		children: make([][]byte,0),
	}
}

func (bn *BlockNode) Hash() []byte {
	return bn.hash
}

func (bn *BlockNode) Parent() []byte {
	return bn.parent
}

func (bn *BlockNode) Depth() uint64 {
	return bn.depth
}

func (bn *BlockNode) Children() [][]byte {
	return bn.children
}

func (bn *BlockNode) AddChild(childHash []byte) int {
	bn.children = append(bn.children, childHash)
	return len(bn.children)
}
