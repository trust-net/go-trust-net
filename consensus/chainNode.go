package consensus

import (
	"github.com/trust-net/go-trust-net/core"
	"encoding/gob"
)

type chainNode struct {
	Parent *core.Byte64
	Depth uint64
	Weight uint64
	IsMainList bool
	Children	 []*core.Byte64
	Hash *core.Byte64
}

func init() {
	gob.Register(&chainNode{})
}

func newChainNode(block *block) *chainNode {
	return &chainNode{
		Parent: block.ParentHash(),
		Hash: block.Hash(),
		Weight: block.Weight().Uint64(),
		Depth: block.Depth().Uint64(),
		IsMainList: false,
		Children: make([]*core.Byte64,0),
	}
}

func (bn *chainNode) isMainList() bool{
	return bn.IsMainList
}

func (bn *chainNode) setMainList(isMainList bool) {
	bn.IsMainList = isMainList
}

func (bn *chainNode) hash() *core.Byte64 {
	return bn.Hash
}

func (bn *chainNode) parent() *core.Byte64 {
	return bn.Parent
}

func (bn *chainNode) depth() uint64 {
	return bn.Depth
}

func (bn *chainNode) weight() uint64 {
	return bn.Weight
}

func (bn *chainNode) children() []*core.Byte64 {
	return bn.Children
}

func (bn *chainNode) addChild(childHash *core.Byte64) int {
	bn.Children = append(bn.Children, childHash)
	return len(bn.Children)
}
