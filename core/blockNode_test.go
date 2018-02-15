package core

import (
    "testing"
    "time"
    "bytes"
)

//var self = *BytesToByte32([]byte("self"))
//var parent = *BytesToByte32([]byte("parent"))
var child1 = []byte("child1")
var child2 = []byte("child2")

//type myNode struct {}
//
//func (n *myNode) Id() string {
//	return "test node"
//}

func TestNewBlockNode(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := time.Duration(time.Now().UnixNano())
	block := NewSimpleBlock(NewSimpleHeader("previous"), NewSimpleHeader("genesis"), now, myNode)
	node := NewBlockNode(block, 11)
	if bytes.Compare(node.Hash(), block.Hash()) != 0 {
		t.Errorf("Hash: Expected: %d, Actual: %d", block.Hash(), node.Hash())
	}
	if bytes.Compare(node.Parent(), block.Previous().Bytes()) != 0 {
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
	now := time.Duration(time.Now().UnixNano())
	block := NewSimpleBlock(NewSimpleHeader("previous"), NewSimpleHeader("genesis"), now, myNode)
	node := NewBlockNode(block, 11)
	node.AddChild(child1)
	node.AddChild(child2)
	if len(node.Children()) != 2 {
		t.Errorf("Childrens: Expected: %d, Actual: %d", 2, len(node.Children()))
	}
	if bytes.Compare(node.Children()[0], child1) != 0 {
		t.Errorf("Child 1: Expected: %d, Actual: %d", child1, node.Children()[0])
	}
	if bytes.Compare(node.Children()[1], child2) != 0 {
		t.Errorf("Child 2: Expected: %d, Actual: %d", child2, node.Children()[1])
	}
}
