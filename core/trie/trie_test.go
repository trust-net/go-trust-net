package trie

import (
    "testing"
    "github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"
)

func TestMakeHex(t *testing.T) {
	data := []byte{0x01,0x23,0x45,0x60}
	expected := []byte{0,1,2,3,4,5,6,0}
	actual := makeHex(data)
	if len(actual) != len(expected) {
		t.Errorf("Incorrect hex conversion: Expected: %x, Actual: %x", expected, actual)
		return
	}
	for i,b := range actual {
		if b != expected[i] {
			t.Errorf("Incorrect hex conversion: Expected: %x, Actual: %x", data, actual)
			return
		}
	}
}

func TestEmptyMptWorldState(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
//	ws, err := NewMptWorldState(db)
//	if err != nil {
//		t.Errorf("Failed to create instance: %s", err)
//		return
//	}
	testEmptyWorldState(t, NewMptWorldState(db))
}

func testEmptyWorldState(t *testing.T, ws WorldState) {
	if ws == nil {
		t.Errorf("Failed to create instance")
	}
	// create an empty node
	node := node{}
	// world state root hash should match empty node's hash
	if ws.Hash() != *node.hash() {
		t.Errorf("Incorrect root hash: Expected: %x, Actual: %x", *node.hash(), ws.Hash())
	}
}

func TestMptWorldStateInsert(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	testWorldStateInsert(t, NewMptWorldState(db))
}

func testWorldStateInsert(t *testing.T, ws WorldState) {
	origHash := ws.Hash()
	// insert some key/value pair into world state
	newHash := ws.Update([]byte("key"), []byte("value"))
	// world state hash should have changed after insert
	if origHash == newHash {
		t.Errorf("World state hash did not change after insert")
	}
	// lets fetch the key
	value, err := ws.Lookup([]byte("key"))
	if err != nil {
		t.Errorf("Lookup of inserted key failed: %s", err)
	}
	if string(value) != "value" {
		t.Errorf("Incorrect lookup: Expected `%s`, Found `%s`", "value", value)		
	}
}

func TestMptWorldStateInvalidLookup(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	testWorldStateInvalidLookup(t, NewMptWorldState(db))
}

func testWorldStateInvalidLookup(t *testing.T, ws WorldState) {
	// lets fetch some non existing key
	_, err := ws.Lookup([]byte("non existing key"))
	if err == nil {
		t.Errorf("Lookup of non existing key did not fail")
	}
}

func TestMptWorldStateDeleteAfterInsert(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	testWorldStateDeleteAfterInsert(t, NewMptWorldState(db))
}

func testWorldStateDeleteAfterInsert(t *testing.T, ws WorldState) {
	// insert some key/value pair into world state
	ws.Update([]byte("key"), []byte("value"))
	// lets fetch the key
	_, err := ws.Lookup([]byte("key"))
	if err != nil {
		t.Errorf("Lookup of inserted key failed: %s", err)
	}
	// now lets delete the key
	origHash := ws.Hash()
	ws.Delete([]byte("key"))
	if origHash == ws.Hash() {
		t.Errorf("World state hash did not change after delete")
	}
	// lookup of deleted key should fail
	_, err = ws.Lookup([]byte("key"))
	if err == nil {
		t.Errorf("Lookup of deleted key did not fail")
	}
}

func TestMptWorldStateDeleteNotExisting(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	testWorldStateDeleteNotExisting(t, NewMptWorldState(db))
}

func testWorldStateDeleteNotExisting(t *testing.T, ws WorldState) {
	// insert some key/value pair into world state
	ws.Update([]byte("key"), []byte("value"))
	// lets fetch the key
	_, err := ws.Lookup([]byte("key"))
	if err != nil {
		t.Errorf("Lookup of inserted key failed: %s", err)
	}
	// now lets delete a non existing key
	origHash := ws.Hash()
	ws.Delete([]byte("non existing key"))
	if origHash != ws.Hash() {
		t.Errorf("World state hash changed after non existing key deletion: Expected %x, Found %x", origHash, ws.Hash())
	}
}


func TestMptWorldStateRebase(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	testWorldStateRebase(t, NewMptWorldState(db))
}

func testWorldStateRebase(t *testing.T, ws WorldState) {
	// insert some key/value pair into world state
	ws.Update([]byte("key1"), []byte("value1"))
	ws.Update([]byte("deleteMe"), []byte("to be deleted"))
	lastState := ws.Hash()
	// not lets update the key/value to change world state
	ws.Update([]byte("key1"), []byte("value2"))
	ws.Update([]byte("key2"), []byte("another value"))
	ws.Delete([]byte("deleteMe"))
	// lets fetch the keys to make sure they are correctly upserted
	value, err := ws.Lookup([]byte("key1"))
	if err != nil {
		t.Errorf("Lookup of updated key1 failed: %s", err)
	}
	if string(value) != "value2" {
		t.Errorf("Incorrect update: Expected `%s`, Found `%s`", "value2", value)		
	}
	value, err = ws.Lookup([]byte("key2"))
	if err != nil {
		t.Errorf("Lookup of updated key2 failed: %s", err)
	}
	if string(value) != "another value" {
		t.Errorf("Incorrect update: Expected `%s`, Found `%s`", "another value", value)		
	}
	_, err = ws.Lookup([]byte("deleteMe"))
	if err == nil {
		t.Errorf("Lookup of deleted key did not fail")
	}
	// lets rebase to older state
	if err = ws.Rebase(lastState); err != nil {
		t.Errorf("Rebase failed: %s", err)
	}
	// lets query world state to make sure it has been reverted to last state
	value, err = ws.Lookup([]byte("key1"))
	if err != nil {
		t.Errorf("Lookup of original key1 failed: %s", err)
	}
	if string(value) != "value1" {
		t.Errorf("Incorrect rebased value: Expected `%s`, Found `%s`", "value1", value)
	}
	_, err = ws.Lookup([]byte("key2"))
	if err == nil {
		t.Errorf("Lookup of non existing key in last state did not fail")
	}
	value, err = ws.Lookup([]byte("deleteMe"))
	if err != nil {
		t.Errorf("Lookup of original deleteMe failed: %s", err)
	}
	if string(value) != "to be deleted" {
		t.Errorf("Incorrect rebased value: Expected `%s`, Found `%s`", "to be deleted", value)		
	}
}

func TestMptWorldStateRebaseNotExisting(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	testWorldStateRebaseNotExisting(t, NewMptWorldState(db))
}

func testWorldStateRebaseNotExisting(t *testing.T, ws WorldState) {
	// try rebasing to some non existing state
	if err := ws.Rebase(*core.BytesToByte64([]byte("invalid state"))); err == nil {
		t.Errorf("Rebase to invalid state did not fail")
	}
}
