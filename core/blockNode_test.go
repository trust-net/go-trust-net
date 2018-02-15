package core

import (
    "testing"
)

var child1 = BytesToByte64([]byte("child1"))
var child2 = BytesToByte64([]byte("child2"))

func TestNewBlockNode(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(BytesToByte64([]byte("previous")), BytesToByte64([]byte("genesis")), myNode)
	node := NewBlockNode(block, 11)
	if node.Hash() != block.Hash() {
		t.Errorf("Hash: Expected: %d, Actual: %d", block.Hash(), node.Hash())
	}
	if node.Parent() != block.Previous().Bytes() {
		t.Errorf("Parent: Expected: %d, Actual: %d", block.Previous().Bytes(), node.Parent())
	}
	if node.Depth() != 11 {
		t.Errorf("Depth: Expected: %d, Actual: %d", 11, node.Depth())
	}
	if len(node.Children()) != 0 {
		t.Errorf("Childrens: Expected: %d, Actual: %d", 0, len(node.Children()))
	}
}

func TestAddChild(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(BytesToByte64([]byte("previous")), BytesToByte64([]byte("genesis")), myNode)
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
