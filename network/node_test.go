package network

import (
    "testing"
    "time"
    "fmt"
    "github.com/trust-net/go-trust-net/db"
)

func timeout(sec time.Duration, done chan int, callback func()) {
	wait := time.NewTimer(sec * time.Second)
	defer wait.Stop()
	for {
		select {
			case <- wait.C:
				fmt.Printf("Timed out after %d seconds\n", sec)
				callback()
			case <- done:
				return
		}
	}
}

func TestPeerNodeToNodeCast(t *testing.T) {
	var node db.PeerNode
	node = NewPeerNode(nil,nil)
	fmt.Printf("peerNode type: %T\n", node)
	if nodeImpl, ok := node.(*peerNode); ok {
		fmt.Printf("node type: %T\n", nodeImpl)
	} else {
		t.Errorf("Failed to cast PeerNode into Node")
	}
}

func TestAddTxAfterMax(t *testing.T) {
	node := NewPeerNode(nil, nil)
	// fill up to capacity
	for i:=0; i < maxKnownTxs; i++ {
		node.AddTx(i)
	}
	// now add one more
	node.AddTx("tx")
	// verify that we did not go over capacity
	if node.knownTxs.Size() >= maxKnownTxs {
		t.Errorf("Expected Size: %d, Actual: %d", maxKnownTxs, node.knownTxs.Size())
	}
	// make sure our new transaction is in there
	if !node.knownTxs.Has("tx") {
		t.Errorf("Expected but did not find: %s", "tx")
	}
}


func TestHasTxAfterAdd(t *testing.T) {
	node := NewPeerNode(nil, nil)
	node.AddTx("tx")
	if !node.HasTx("tx") {
		t.Errorf("Expected but did not find: %s", "tx")
	}
}

func TestNoHasTxUnKnown(t *testing.T) {
	node := NewPeerNode(nil, nil)
	node.AddTx("tx1")
	if node.HasTx("tx2") {
		t.Errorf("Not Expected but did find: %s", "tx2")
	}
}

func TestNoHasTxEmptySet(t *testing.T) {
	node := NewPeerNode(nil, nil)
	if node.HasTx("tx") {
		t.Errorf("Not Expected but did find: %s", "tx")
	}
}

func TestAddTxUnlocked(t *testing.T) {
	node := NewPeerNode(nil, nil)
	// hack, force node's lock to be locked, so it will timeout and bark if tried to lock
	node.lock.Lock()
	flag := true
	defer func(){
		if flag {
			node.lock.Unlock()
		} else {
			fmt.Printf("Skipping unlock!!!\n")
		}
	}()
	done := make(chan int)
	go timeout(5, done, func(){
			t.Errorf("Failed to add transaction")
			node.lock.Unlock()
			flag = false
		})
	fmt.Printf("Calling add transaction...\n")
	node.AddTx("tx1")
	fmt.Printf("Back after adding transaction\n")
	done <- 1
}

func TestAddBlockAfterMax(t *testing.T) {
	node := NewPeerNode(nil, nil)
	// fill up to capacity
	for i:=0; i < maxKnownBlocks; i++ {
		node.AddBlock(i)
	}
	// now add one more
	node.AddBlock("block")
	// verify that we did not go over capacity
	if node.knownBlocks.Size() >= maxKnownBlocks {
		t.Errorf("Expected Size: %d, Actual: %d", maxKnownBlocks, node.knownBlocks.Size())
	}
	// make sure our new block is in there
	if !node.knownBlocks.Has("block") {
		t.Errorf("Expected but did not find: %s", "block")
	}
}


func TestHasBlockAfterAdd(t *testing.T) {
	node := NewPeerNode(nil, nil)
	node.AddBlock("block")
	if !node.HasBlock("block") {
		t.Errorf("Expected but did not find: %s", "block")
	}
}

func TestNoHasBlockUnKnown(t *testing.T) {
	node := NewPeerNode(nil, nil)
	node.AddBlock("block1")
	if node.HasBlock("block2") {
		t.Errorf("Not Expected but did find: %s", "block2")
	}
}

func TestNoHasBlockEmptySet(t *testing.T) {
	node := NewPeerNode(nil, nil)
	if node.HasBlock("block") {
		t.Errorf("Not Expected but did find: %s", "block")
	}
}

func TestAddBlockUnlocked(t *testing.T) {
	node := NewPeerNode(nil, nil)
	// hack, force node's lock to be locked, so it will timeout and bark if tried to lock
	node.lock.Lock()
	flag := true
	defer func(){
		if flag {
			node.lock.Unlock()
		} else {
			fmt.Printf("Skipping unlock!!!\n")
		}
	}()
	done := make(chan int)
	go timeout(5, done, func(){
			t.Errorf("Failed to add block")
			node.lock.Unlock()
			flag = false
		})
	fmt.Printf("Calling add block...\n")
	node.AddBlock("tx1")
	fmt.Printf("Back after adding block\n")
	done <- 1
}
