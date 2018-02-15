package core

import (
	"time"

)

type BlockChain interface {
	
}

type BlockChainInMem struct {
	genesis *BlockNode
	leaves []*BlockNode
	TD	time.Duration
	depth uint64
}

func NewBlockChainInMem() *BlockChainInMem {
//	genesis := &SimpleBlock{}
	genesis := NewSimpleBlock(NewSimpleHeader(""), NewSimpleHeader(""), time.Duration(time.Now().UnixNano()), NewSimpleNodeInfo(""))
	return &BlockChainInMem{
		genesis: NewBlockNode(genesis, 0),
		depth: 0,
		TD: genesis.TD(),
		leaves: make([]*BlockNode, 0),
	}
}
