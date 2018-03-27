package trie

import (
    "testing"
    "github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/log"
)


func TestTxEmptyMptWorldState(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	testTxEmptyWorldState(t, NewMptWorldState(db))
}

func testTxEmptyWorldState(t *testing.T, ws WorldState) {
	if ws == nil {
		t.Errorf("Failed to create instance")
	}
	// create an empty node
	node := node{}
	// world state root hash should match empty node's hash
	if ws.Hash() != *node.hash() {
		t.Errorf("Incorrect root hash: Expected: %x, Actual: %x", *node.hash(), ws.Hash())
	}
	// world state should not have any transaction
	if _, err := ws.HasTransaction(node.hash()); err == nil {
		t.Errorf("empty world state should not have any transactions")
	}
}

func TestTxMptWorldStateRegister(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	testTxWorldStateRegister(t, NewMptWorldState(db))
}

func testTxWorldStateRegister(t *testing.T, ws WorldState) {
	origHash := ws.Hash()
	// register a transaction
	txId := core.BytesToByte64([]byte("some random transaction"))
	blockId := core.BytesToByte64([]byte("some random block"))
	if err:= ws.RegisterTransaction(txId, blockId); err != nil {
		t.Errorf("Failed to register transaction: %s", err)
	}
	// world state hash should not have changed after registering transaction
	if origHash != ws.Hash() {
		t.Errorf("World state hash changed when registering transaction")
	}
	// lets check if transaction exists
	if fetchedId, err := ws.HasTransaction(txId); err != nil {
		t.Errorf("Failed to check transaction: %s", err)
	} else if fetchedId == nil {
		t.Errorf("Did not find transaction: %x", txId)
	} else if *fetchedId != *blockId {
		t.Errorf("Transaction's block ID incorrect:\nExpected %x\nFound %x", *blockId, *fetchedId)
	}
}

func TestTxWorldStateCleanupAfterRebalance(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	testTxWorldStateCleanupAfterRebalance(t, NewMptWorldState(db))
}

func testTxWorldStateCleanupAfterRebalance(t *testing.T, ws WorldState) {
	origHash := ws.Hash()
	// update world state and register two transactions with first block
	txId_1 := core.BytesToByte64([]byte("transaction #1"))
	txId_2 := core.BytesToByte64([]byte("transaction #2"))
	blockId_1 := core.BytesToByte64([]byte("block #1"))
	ws.Update([]byte("key"), []byte("value1"))
	ws.RegisterTransaction(txId_1, blockId_1)
	ws.RegisterTransaction(txId_2, blockId_1)
	staleHash := ws.Hash()

	// now simulate a canonical change rebalance and register only first transaction with different block
	ws.Rebase(origHash)
	blockId_2 := core.BytesToByte64([]byte("block #2"))
	ws.Update([]byte("key"), []byte("value2"))
	if err:= ws.RegisterTransaction(txId_1, blockId_2); err != nil {
		t.Errorf("Failed to register transaction: %s", err)
	}

	// verify that first transaction is associated with new block after rebalance
	if fetchedId, err := ws.HasTransaction(txId_1); err != nil {
		t.Errorf("Failed to check transaction: %s", err)
	} else if fetchedId == nil {
		t.Errorf("Did not find transaction: %x", txId_1)
	} else if *fetchedId != *blockId_2 {
		t.Errorf("Transaction's block ID incorrect:\nExpected %x\nFound %x", *blockId_2, *fetchedId)
	}

	// verify that second transaction is not available after rebalance
	if _, err := ws.HasTransaction(txId_2); err == nil {
		t.Errorf("did not expect 2nd transaction after rebalance")
	}
	// verify that old transaction trie is still in the DB after rebalance
	if _, found := ws.(*MptWorldState).getNode(tableKey(tableTransactionRootByWorldState, &staleHash)); !found {
		t.Errorf("did not find old transaction trie in DB after rebalance")
	}

	// cleanup the world state
	if err := ws.Cleanup(staleHash); err != nil {
		t.Errorf("Failed to cleanup: %s", err)
	}
	// verify that sold transaction trie is cleaned up from DB
	if _, found := ws.(*MptWorldState).getNode(tableKey(tableTransactionRootByWorldState, &staleHash)); found {
		t.Errorf("did not expect old transaction trie in DB after cleanup")
	}
	// verify that first transaction is still available after cleanup
	if _, err := ws.HasTransaction(txId_1); err != nil {
		t.Errorf("Failed to check transaction after cleanup: %s", err)
	}
}

func TestTxWorldStateRegisterBeforeStateUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	testTxWorldStateRegisterBeforeStateUpdate(t, NewMptWorldState(db))
}
func testTxWorldStateRegisterBeforeStateUpdate(t *testing.T, ws WorldState) {
	// register transaction before world state change
	txId := core.BytesToByte64([]byte("some random transaction"))
	blockId_1 := core.BytesToByte64([]byte("block #1"))
	ws.RegisterTransaction(txId, blockId_1)
	// now cause a chance in the world state
	ws.Update([]byte("key"), []byte("value1"))
	ws.Update([]byte("key"), []byte("value2"))
	
	// verify that transaction is still accessible after world state change
	if fetchedId, err := ws.HasTransaction(txId); err != nil {
		t.Errorf("Failed to check transaction: %s", err)
	} else if fetchedId == nil {
		t.Errorf("Did not find transaction: %x", txId)
	} else if *fetchedId != *blockId_1 {
		t.Errorf("Transaction's block ID incorrect:\nExpected %x\nFound %x", *blockId_1, *fetchedId)
	}
}
