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

func (cn *chainNode) isMainList() bool{
	return cn.IsMainList
}

func (cn *chainNode) setMainList(isMainList bool) {
	cn.IsMainList = isMainList
}

func (cn *chainNode) hash() *core.Byte64 {
	return cn.Hash
}

func (cn *chainNode) parent() *core.Byte64 {
	return cn.Parent
}

func (cn *chainNode) depth() uint64 {
	return cn.Depth
}

func (cn *chainNode) weight() uint64 {
	return cn.Weight
}

func (cn *chainNode) children() []*core.Byte64 {
	return cn.Children
}

func (cn *chainNode) addChild(childHash *core.Byte64) int {
	cn.Children = append(cn.Children, childHash)
	return len(cn.Children)
}
