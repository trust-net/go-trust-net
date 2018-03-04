package chain

import (
    "testing"
	"github.com/trust-net/go-trust-net/core"
)

var child1 = core.BytesToByte64([]byte("child1"))
var child2 = core.BytesToByte64([]byte("child2"))

func TestNewBlockNode(t *testing.T) {
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(core.BytesToByte64([]byte("previous")), 0, myNode)
	block.ComputeHash()
	node := NewBlockNode(block, 11)
	if *node.Hash() != *block.Hash() {
		t.Errorf("Hash: Expected: %d, Actual: %d", block.Hash(), node.Hash())
	}
	if node.Parent() != block.ParentHash() {
		t.Errorf("Parent: Expected: %d, Actual: %d", block.ParentHash(), node.Parent())
	}
	if node.Depth() != 11 {
		t.Errorf("Depth: Expected: %d, Actual: %d", 11, node.Depth())
	}
	if len(node.Children()) != 0 {
		t.Errorf("Childrens: Expected: %d, Actual: %d", 0, len(node.Children()))
	}
	if node.IsMainList() {
		t.Errorf("Is main list: Expected: %d, Actual: %d", false, node.IsMainList())
	}
	if *node.Block() != *block.Hash() {
		t.Errorf("Block: Expected: %d, Actual: %d", block.Hash(), node.Block())
	}
}

func TestSetIsMain(t *testing.T) {
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(core.BytesToByte64([]byte("previous")), 0, myNode)
	block.ComputeHash()
	node := NewBlockNode(block, 11)
	if node.IsMainList() {
		t.Errorf("Is main list: Expected: %d, Actual: %d", false, node.IsMainList())
	}
	node.SetMainList(true)
	if !node.IsMainList() {
		t.Errorf("Is main list: Expected: %d, Actual: %d", false, node.IsMainList())
	}	
}

func TestAddChild(t *testing.T) {
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(core.BytesToByte64([]byte("previous")), 0, myNode)
	block.ComputeHash()
	node := NewBlockNode(block, 11)
	node.AddChild(child1)
	node.AddChild(child2)
	if len(node.Children()) != 2 {
		t.Errorf("Childrens: Expected: %d, Actual: %d", 2, len(node.Children()))
	}
	if node.Children()[0] != child1 {
		t.Errorf("Child 1: Expected: %d, Actual: %d", child1, node.Children()[0])
	}
	if node.Children()[1] != child2 {
		t.Errorf("Child 2: Expected: %d, Actual: %d", child2, node.Children()[1])
	}
}

type testError struct {}

func (e *testError) Error() string {
	return ""
}
