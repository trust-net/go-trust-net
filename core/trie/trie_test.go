package trie

import (
    "testing"
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

func TestEmptyInMemWorldState(t *testing.T) {
	testEmptyWorldState(t, NewInMemWorldState())
}

func testEmptyWorldState(t *testing.T, ws WorldState) {
	// create an empty node
	node := node{}
	// world state root hash should match empty node's hash
	if ws.Hash() != *node.hash() {
		t.Errorf("Incorrect root hash: Expected: %x, Actual: %x", *node.hash(), ws.Hash())
	}
}

func TestInMemWorldStateInsert(t *testing.T) {
	testWorldStateInsert(t, NewInMemWorldState())
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

func TestInMemWorldStateInvalidLookup(t *testing.T) {
	testWorldStateInvalidLookup(t, NewInMemWorldState())
}

func testWorldStateInvalidLookup(t *testing.T, ws WorldState) {
	// lets fetch some non existing key
	_, err := ws.Lookup([]byte("non existing key"))
	if err == nil {
		t.Errorf("Lookup of non existing key did not fail")
	}
}

func TestInMemWorldStateDeleteAfterInsert(t *testing.T) {
	testWorldStateDeleteAfterInsert(t, NewInMemWorldState())
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

func TestInMemWorldStateDeleteNotExisting(t *testing.T) {
	testWorldStateDeleteNotExisting(t, NewInMemWorldState())
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
