package trie

import (
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/core"
)

var (
	null = *core.BytesToByte64(nil)
)

type node struct {
	idxs [16] core.Byte64
	value []byte
}

func (n *node) hash() *core.Byte64 {
	data := make([]byte, 16*64 + len(n.value))
	for i:=0; i<16; i++ {
		data = append(data, n.idxs[i].Bytes()...)
	}
	data = append(data, n.value...)
	hash := sha512.Sum512(data)
	return core.BytesToByte64(hash[:])
}

// an in memory implementation (can be used as cache, wrapped with persistent implementation on top)
// this implementation uses 128KB memory for a single entry of 64 byte key
// (16 indexes * 64 byte/index * 1 key * 64 byte/key * 2 nibbles/byte = 128k bytes)
//
// also, each lookup will take exactly 128 reads of the DB
// (64 byte "key length" * 2 nibble/byte * 1 "next node" lookup/nibble = 128 lookup)
type InMemWorldState struct {
	root core.Byte64
	db map[core.Byte64]node
}

func NewInMemWorldState() WorldState {
	ws := InMemWorldState {
		db: make(map[core.Byte64]node),
	}
	rootNode := node{}
	ws.root = *rootNode.hash()
	ws.db[ws.root] = rootNode
	return &ws
}

func makeHex(data []byte) []byte {
	l := len(data)*2
	var nibbles = make([]byte, l)
	for i, b := range data {
		nibbles[i*2] = b / 0x10
		nibbles[i*2+1] = b % 0x10
	}
	return nibbles
}

func (t *InMemWorldState) Update(key, value []byte) core.Byte64 {
	// convert key into hexadecimal nibbles
	keyHex := makeHex(key)
	t.root = t.updateDepthFirst(t.db[t.root], keyHex, value)
	return t.root
}

func (t *InMemWorldState) updateDepthFirst(currNode node, keyHex, value []byte) core.Byte64 {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		// update the node value
		currNode.value = value
		// save the node into db
		newHash := *currNode.hash()
		t.db[newHash] = currNode
		return newHash
	}
	
	// else continue to update depth first, and then update current node with child's hash
	childHash := currNode.idxs[keyHex[0]]
	childNode := node{}
	if childHash != null {
		childNode = t.db[childHash]
	}
	childHash = t.updateDepthFirst(childNode, keyHex[1:], value)
	// update current node's key index with new hash of child node
	currNode.idxs[keyHex[0]] = childHash
	// recompute hash for the current node after child hash update
	newHash := *currNode.hash()
	// save the node into db
	t.db[newHash] = currNode
	// return back recomputed hash of current node
	return newHash
}

func (t *InMemWorldState) Delete(key []byte) core.Byte64 {
	// convert key into hexadecimal nibbles
	keyHex := makeHex(key)
	if newHash := t.deleteDepthFirst(t.db[t.root], keyHex); newHash != null {
		t.root = newHash
	}
	return t.root
}

func (t *InMemWorldState) deleteDepthFirst(currNode node, keyHex []byte) core.Byte64 {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		// delete the node value
		currNode.value = nil
		// save the node into db
		newHash := *currNode.hash()
		t.db[newHash] = currNode
		return newHash
	}
	
	// else continue to delte depth first, and then update current node with child's hash
	childHash := currNode.idxs[keyHex[0]]
	if childHash != null {
		childNode := t.db[childHash]
		childHash = t.deleteDepthFirst(childNode, keyHex[1:])
		// if the depth first delete did not change anything, return as is
		if childHash == null {
			return null
		}
		// update current node's key index with new hash of child node
		currNode.idxs[keyHex[0]] = childHash
		// recompute hash for the current node after child hash update
		newHash := *currNode.hash()
		// save the node into db
		t.db[newHash] = currNode
		// return back recomputed hash of current node
		return newHash
	} else {
		return null
	}	
}

func (t *InMemWorldState) Lookup(key []byte) ([]byte, error) {
	// convert key into hexadecimal nibbles
	keyHex := makeHex(key)
	if value := t.lookupDepthFirst(t.db[t.root], keyHex); len(value) == 0 {
		return nil, core.NewCoreError(ERR_NOT_FOUND, "key does not exists")
	} else {
		return value, nil
	}
}

func (t *InMemWorldState) lookupDepthFirst(currNode node, keyHex []byte) []byte {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		return currNode.value
	}
	// else continue to lookup depth first
	if childHash := currNode.idxs[keyHex[0]]; childHash != null {
		childNode := t.db[childHash]
		return t.lookupDepthFirst(childNode, keyHex[1:])
	} else {
		return nil
	}	
}

func (t *InMemWorldState) Hash() core.Byte64 {
	return t.root
}

func (t *InMemWorldState) Rebase(hash core.Byte64) error {
	// lookup node for the provided hash
	if _, ok := t.db[hash]; !ok {
		return core.NewCoreError(ERR_NOT_FOUND, "key does not exists")
	}
	// node exists, so we can rebase
	t.root = hash
	return nil
}
