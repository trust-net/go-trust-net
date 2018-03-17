package trie

import (
	"sync"
	"crypto/sha512"
	"encoding/gob"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/common"
)

var (
	null = *core.BytesToByte64(nil)
)

type node struct {
	Idxs [16] core.Byte64
	Value []byte
}


var tableMptWorldStateNode = []byte("MptWorldStateNode-")
func tableKey(prefix []byte, key *core.Byte64) []byte {
	return append(prefix, key.Bytes()...)
}

func init() {
	gob.Register(&node{})
}

func (n *node) hash() *core.Byte64 {
	data := make([]byte, 16*64 + len(n.Value))
	for i:=0; i<16; i++ {
		data = append(data, n.Idxs[i].Bytes()...)
	}
	data = append(data, n.Value...)
	hash := sha512.Sum512(data)
	return core.BytesToByte64(hash[:])
}

// an in memory implementation (can be used as cache, wrapped with persistent implementation on top)
// this implementation uses 128KB memory for a single entry of 64 byte key
// (16 indexes * 64 byte/index * 1 key * 64 byte/key * 2 nibbles/byte = 128k bytes)
//
// also, each lookup will take exactly 128 reads of the DB
// (64 byte "key length" * 2 nibble/byte * 1 "next node" lookup/nibble = 128 lookup)
type MptWorldState struct {
	rootHash *core.Byte64
	rootNode *node
	db db.Database
	lock sync.RWMutex
	logger log.Logger
}

func NewMptWorldState(db db.Database) WorldState {
	ws := MptWorldState {
		db: db,
		rootNode: &node{},
	}
	ws.logger = log.NewLogger(ws)
	ws.rootHash = ws.rootNode.hash()
	if err := ws.put(ws.rootHash, ws.rootNode); err != nil {
		ws.logger.Error("Failed to initialize: %s", err)
		return nil
	}
	return &ws
}

func (t *MptWorldState) put(hash *core.Byte64, node *node) error {
	if data, err := common.Serialize(node); err == nil {
		return t.db.Put(tableKey(tableMptWorldStateNode, hash), data)
	} else {
		t.logger.Error("failed to encode data for DB: %s", err.Error())
		return err
	}
}

func (t *MptWorldState) get(hash *core.Byte64) (*node, bool) {
	if data, err := t.db.Get(tableKey(tableMptWorldStateNode, hash)); err != nil {
		t.logger.Error("Did not find node in DB: %s", err.Error())
		return nil, false
	} else {
		var node node
		if err := common.Deserialize(data, &node); err != nil {
			t.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, false
		}
		return &node, true
	}
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

func (t *MptWorldState) Update(key, value []byte) core.Byte64 {
	// get lock
	t.lock.Lock()
	defer t.lock.Unlock()
	// convert key into hexadecimal nibbles
	keyHex := makeHex(key)
	if newHash := t.updateDepthFirst(t.rootNode, keyHex, value); newHash != null {
//		if newNode, ok := t.get(&newHash); ok {
			t.rootHash = &newHash
//			t.rootNode = newNode
//		}
	}
	return *t.rootHash
}

func (t *MptWorldState) updateDepthFirst(currNode *node, keyHex, value []byte) core.Byte64 {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		// update the node value
		currNode.Value = value
		// save the node into db
		newHash := currNode.hash()
		if err := t.put(newHash, currNode); err != nil {
			t.logger.Error("Failed to update node: %s", err)
			return null
		}
		return *newHash
	}
	
	// else continue to update depth first, and then update current node with child's hash
	childHash := currNode.Idxs[keyHex[0]]
	childNode := &node{}
	if childHash != null {
		if childNode,_ = t.get(&childHash); childNode == nil {
			childNode = &node{}
		}
	}
	childHash = t.updateDepthFirst(childNode, keyHex[1:], value)
	if childHash == null {
		return null
	}
	// update current node's key index with new hash of child node
	currNode.Idxs[keyHex[0]] = childHash
	// recompute hash for the current node after child hash update
	newHash := *currNode.hash()
	// save the node into db
	if err := t.put(&newHash, currNode); err != nil {
		t.logger.Error("Failed to update node: %s", err)
		return null
	}
	// return back recomputed hash of current node
	return newHash
}

func (t *MptWorldState) Delete(key []byte) core.Byte64 {
	// get lock
	t.lock.Lock()
	defer t.lock.Unlock()
	// convert key into hexadecimal nibbles
	keyHex := makeHex(key)
	if newHash := t.deleteDepthFirst(t.rootNode, keyHex); newHash != null {
//		if newNode, ok := t.get(&newHash); ok {
			t.rootHash = &newHash
//			t.rootNode = newNode
//		}
	}
	return *t.rootHash
}

func (t *MptWorldState) deleteDepthFirst(currNode *node, keyHex []byte) core.Byte64 {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		// delete the node value
		currNode.Value = nil
		// save the node into db
		newHash := currNode.hash()
		if err := t.put(newHash, currNode); err != nil {
			t.logger.Error("Failed to update node: %s", err)
			return null
		}
		return *newHash
	}
	
	// else continue to delte depth first, and then update current node with child's hash
	childHash := currNode.Idxs[keyHex[0]]
	if childHash != null {
		childNode, _ := t.get(&childHash)
		if childNode == nil {
			return null
		}
		childHash = t.deleteDepthFirst(childNode, keyHex[1:])
		// if the depth first delete did not change anything, return as is
		if childHash == null {
			return null
		}
		// update current node's key index with new hash of child node
		currNode.Idxs[keyHex[0]] = childHash
		// recompute hash for the current node after child hash update
		newHash := *currNode.hash()
		// save the node into db
		if err := t.put(&newHash, currNode); err != nil {
			t.logger.Error("Failed to update node: %s", err)
			return null
		}
		// return back recomputed hash of current node
		return newHash
	} else {
		return null
	}	
}

func (t *MptWorldState) Lookup(key []byte) ([]byte, error) {
	// get lock
	t.lock.RLock()
	defer t.lock.RUnlock()
	// convert key into hexadecimal nibbles
	keyHex := makeHex(key)
	if value := t.lookupDepthFirst(t.rootNode, keyHex); len(value) == 0 {
		return nil, core.NewCoreError(ERR_NOT_FOUND, "key does not exists")
	} else {
		return value, nil
	}
}

func (t *MptWorldState) lookupDepthFirst(currNode *node, keyHex []byte) []byte {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		return currNode.Value
	}
	// else continue to lookup depth first
	if childHash := currNode.Idxs[keyHex[0]]; childHash != null {
        if childNode, found := t.get(&childHash); found {
	        	return t.lookupDepthFirst(childNode, keyHex[1:])
        } else {
	        	return nil
        }
	} else {
		return nil
	}	
}

func (t *MptWorldState) Hash() core.Byte64 {
	return *t.rootHash
}

func (t *MptWorldState) Rebase(hash core.Byte64) error {
	// lookup node for the provided hash
	if rootNode, ok := t.get(&hash); !ok {
		t.logger.Error("Failed to rebase world state to hash %x", hash)
		return core.NewCoreError(ERR_NOT_FOUND, "hash does not exists")
	} else {
		// node exists, so we can rebase
		t.rootHash = &hash
		t.rootNode = rootNode
		t.logger.Error("Rebased world state to hash %x", hash)
	}
	return nil
}
