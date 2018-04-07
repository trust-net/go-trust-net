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
	uninitialized = byte(0x00)
	marked = byte(0x01)
	unmarked = byte(0x02)
)

type node struct {
	Idxs [16] core.Byte64
	Value []byte
	Tombstone byte
}

var tableTransactionRootByWorldState = []byte("TransactionRootByWorldState-")
var tableTransactionNode = []byte("MptTransactionNode-")
var tableTransactionFlatKV = []byte("MptTransactionFlatKV-")
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
	txNode *node
	db db.Database
	lock sync.RWMutex
	logger log.Logger
}

func NewMptWorldState(db db.Database) WorldState {
	ws := MptWorldState {
		db: db,
		rootNode: &node{
			Tombstone: uninitialized,
		},
		txNode: &node{
			Tombstone: uninitialized,
		},
	}
	ws.logger = log.NewLogger(ws)
	ws.rootHash = ws.rootNode.hash()
	if err := ws.putState(ws.rootHash, ws.rootNode); err != nil {
		ws.logger.Error("Failed to initialize: %s", err)
		return nil
	}
	if err := ws.putNode(tableKey(tableTransactionRootByWorldState, ws.rootHash), ws.txNode); err != nil {
		ws.logger.Error("failed to initialize transaction trie: %s", err.Error())
		return nil
	}
	return &ws
}

func (t *MptWorldState) putNode(key []byte, node *node) error {
	if data, err := common.Serialize(node); err == nil {
		return t.db.Put(key, data)
	} else {
		t.logger.Error("failed to encode data for DB: %s", err.Error())
		return err
	}
}

type putter func(hash *core.Byte64, node *node) error

func (t *MptWorldState) putState(hash *core.Byte64, node *node) error {
	return t.putNode(tableKey(tableMptWorldStateNode, hash), node)
}

func (t *MptWorldState) putTx(hash *core.Byte64, node *node) error {
	return t.putNode(tableKey(tableTransactionNode, hash), node)
}

func (t *MptWorldState) getNode(key []byte) (*node, bool) {
	if data, err := t.db.Get(key); err != nil {
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

type deleter func(hash *core.Byte64) error

func (t *MptWorldState) deleteNode(key []byte) error {
	return t.db.Delete(key)
}

func (t *MptWorldState) deleteTx(hash *core.Byte64) error {
	return t.deleteNode(tableKey(tableTransactionNode, hash))
}


func (t *MptWorldState) deleteState(hash *core.Byte64) error {
	return t.deleteNode(tableKey(tableMptWorldStateNode, hash))
}

type getter func(hash *core.Byte64) (*node, bool)

func (t *MptWorldState) getState(hash *core.Byte64) (*node, bool) {
	return t.getNode(tableKey(tableMptWorldStateNode, hash))
}

func (t *MptWorldState) getTx(hash *core.Byte64) (*node, bool) {
	return t.getNode(tableKey(tableTransactionNode, hash))
}

func (t *MptWorldState) HasTransaction(txId *core.Byte64) (*core.Byte64, error) {
	// get lock
	t.lock.RLock()
	defer t.lock.RUnlock()
//	return t.hasTransactionInTrie(txId)
	return t.hasTransactionInFlatKV(txId)
}

func (t *MptWorldState) hasTransactionInFlatKV(txId *core.Byte64) (*core.Byte64, error) {
	if data, err := t.db.Get(tableKey(tableTransactionFlatKV, txId)); err != nil {
		t.logger.Debug("Did not find transaction in DB: %s", err.Error())
		return nil, err
	} else {
		return core.BytesToByte64(data), nil
	}
}

func (t *MptWorldState) hasTransactionInTrie(txId *core.Byte64) (*core.Byte64, error) {
	// convert key into hexadecimal nibbles
	keyHex := makeHex(txId.Bytes())
	// lookup transaction
	if value := t.lookupDepthFirst(t.getTx, t.txNode, keyHex); len(value) == 0 {
		return nil, core.NewCoreError(ERR_NOT_FOUND, "transaction does not exists")
	} else {
		return core.BytesToByte64(value), nil
	}
}

func (t *MptWorldState) RegisterTransaction(txId, blockId *core.Byte64) error {
	// get lock
	t.lock.Lock()
	defer t.lock.Unlock()
//	return t.registerTransactionToTrie(txId, blockId)
	return t.registerTransactionToFlatKV(txId, blockId)
}

func (t *MptWorldState) registerTransactionToFlatKV(txId, blockId *core.Byte64) error {
	return t.db.Put(tableKey(tableTransactionFlatKV, txId), blockId.Bytes())
}

func (t *MptWorldState) registerTransactionToTrie(txId, blockId *core.Byte64) error {
	// convert key into hexadecimal nibbles
	keyHex := makeHex(txId.Bytes())
	// first get transaction trie root for current world view
	txNodeCopy := *t.txNode
	if newHash := t.updateDepthFirst(t.putTx, t.getTx, &txNodeCopy, keyHex, blockId.Bytes()); newHash != null {		
		// save the new transaction trie toot hash for current world view
		if err := t.putNode(tableKey(tableTransactionRootByWorldState, t.rootHash), &txNodeCopy); err != nil {
			t.logger.Error("failed to update transaction trie: %s", err.Error())
			return core.NewCoreError(ERR_UPDATE_FAILED, err.Error())
		}
		// mark tombstone on old transaction trie root
		t.tombStone(t.putTx, t.txNode)
		t.cleanupDepthFirst(t.deleteTx, t.getTx, t.txNode)
//		t.logger.Debug("Switching transaction trie root:\nOld %x\nNew %x", *t.txNode.hash(), *txNodeCopy.hash())
		// switch to new transaction trie root
		t.txNode = &txNodeCopy
	} else {
			t.logger.Error("failed to update transaction ID: %x", txId)
			return core.NewCoreError(ERR_UPDATE_FAILED, "transaction update failed")
	}
	return nil
}

func (t *MptWorldState) tombStone(put putter, currNode *node) {
	if currNode.Tombstone != uninitialized {
		currNode.Tombstone = marked
		if err := put(currNode.hash(), currNode); err != nil {
			t.logger.Error("failed to mark tombstone on node: %s", err.Error())
		}
//		t.logger.Debug("Maked tombstone on node: %x", *currNode.hash())
	}
	currNode.Tombstone = unmarked
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
	if newHash := t.updateDepthFirst(t.putState, t.getState, t.rootNode, keyHex, value); newHash != null {
		// mark tombstone on old world state view of transaction trie root
		t.txNode.Tombstone = marked
		t.putNode(tableKey(tableTransactionRootByWorldState, t.rootHash), t.txNode)
		t.txNode.Tombstone = unmarked
		// update the transaction trie for new world state
		if err := t.putNode(tableKey(tableTransactionRootByWorldState, &newHash), t.txNode); err != nil {
			t.logger.Error("failed to update transaction trie: %s", err.Error())
			return *t.rootHash
		}
		t.rootHash = &newHash
	}
	return *t.rootHash
}

func (t *MptWorldState) updateDepthFirst(put putter, get getter, currNode *node, keyHex, value []byte) core.Byte64 {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		// first mark the tomstone on the node
		t.tombStone(put, currNode)
		// update the node value
		currNode.Value = value
		// save the node into db
		newHash := currNode.hash()
		if err := put(newHash, currNode); err != nil {
			t.logger.Error("Failed to update node: %s", err)
			return null
		}
		return *newHash
	}
	
	// else continue to update depth first, and then update current node with child's hash
	childHash := currNode.Idxs[keyHex[0]]
//	childNode := &node{}
	var exists bool
	var childNode *node
	if childHash != null {
		childNode,exists = get(&childHash)
	}
	if !exists {
		childNode = &node{
			Tombstone: uninitialized,
		}
	}
	childHash = t.updateDepthFirst(put, get, childNode, keyHex[1:], value)
	if childHash == null {
		return null
	}
	// first mark the tomstone on the node
	t.tombStone(put, currNode)
	// update current node's key index with new hash of child node
	currNode.Idxs[keyHex[0]] = childHash
	// recompute hash for the current node after child hash update
	newHash := *currNode.hash()
	// save the node into db
	if err := put(&newHash, currNode); err != nil {
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
	if newHash := t.deleteDepthFirst(t.putState, t.getState, t.rootNode, keyHex); newHash != null {
		// mark tombstone on old world state view of transaction trie root
		t.txNode.Tombstone = marked
		t.putNode(tableKey(tableTransactionRootByWorldState, t.rootHash), t.txNode)
		t.txNode.Tombstone = unmarked
		// update the transaction trie for new world state
		if err := t.putNode(tableKey(tableTransactionRootByWorldState, &newHash), t.txNode); err != nil {
			t.logger.Error("failed to update transaction trie: %s", err.Error())
			return *t.rootHash
		}
		t.rootHash = &newHash
	}
	return *t.rootHash
}

func (t *MptWorldState) deleteDepthFirst(put putter, get getter, currNode *node, keyHex []byte) core.Byte64 {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		// first mark the tomstone on the node
		t.tombStone(put, currNode)
		// delete the node value
		currNode.Value = nil
		// save the node into db
		newHash := currNode.hash()
		if err := put(newHash, currNode); err != nil {
			t.logger.Error("Failed to update node: %s", err)
			return null
		}
		return *newHash
	}
	
	// else continue to delte depth first, and then update current node with child's hash
	childHash := currNode.Idxs[keyHex[0]]
	if childHash != null {
		childNode, _ := get(&childHash)
		if childNode == nil {
			return null
		}
		childHash = t.deleteDepthFirst(put, get, childNode, keyHex[1:])
		// if the depth first delete did not change anything, return as is
		if childHash == null {
			return null
		}
		// first mark the tomstone on the node
		t.tombStone(put, currNode)
		// update current node's key index with new hash of child node
		currNode.Idxs[keyHex[0]] = childHash
		// recompute hash for the current node after child hash update
		newHash := *currNode.hash()
		// save the node into db
		if err := put(&newHash, currNode); err != nil {
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
	if value := t.lookupDepthFirst(t.getState, t.rootNode, keyHex); len(value) == 0 {
		return nil, core.NewCoreError(ERR_NOT_FOUND, "key does not exists")
	} else {
		return value, nil
	}
}

func (t *MptWorldState) lookupDepthFirst(get getter, currNode *node, keyHex []byte) []byte {
	// check if we have reached the end of key
	if len(keyHex) == 0 {
		return currNode.Value
	}
	// else continue to lookup depth first
	if childHash := currNode.Idxs[keyHex[0]]; childHash != null {
        if childNode, found := get(&childHash); found {
	        	return t.lookupDepthFirst(get, childNode, keyHex[1:])
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
	if rootNode, ok := t.getState(&hash); !ok {
		t.logger.Error("Failed to rebase world state to hash %x", hash)
		return core.NewCoreError(ERR_NOT_FOUND, "hash does not exists")
	} else {
//		t.logger.Debug("Found node to rebase %x", hash)
		// node exists, so find the transaction trie root for this node
		var txNode *node
		if txNode, _ = t.getNode(tableKey(tableTransactionRootByWorldState, &hash)); txNode == nil {
			t.logger.Error("Failed to find transaction trie for the rebase state %x", hash)
			return core.NewCoreError(ERR_NOT_FOUND, "transaction trie does not exists")
		}
		// remove any old tombstone on rebased nodes
		rootNode.Tombstone = unmarked
		if err := t.putState(rootNode.hash(), rootNode); err != nil {
			t.logger.Error("failed to unmark tombstone on rebased node: %s", err.Error())
			return err
		}
		txNode.Tombstone = unmarked
		if err := t.putNode(tableKey(tableTransactionRootByWorldState, rootNode.hash()), txNode); err != nil {
			t.logger.Error("failed to unmark tombstone on rebased node: %s", err.Error())
			return err
		}
		// now mark the tombstone on old world state root node
		t.tombStone(t.putState, t.rootNode)
		// mark tombstone on old world state view of transaction trie root
		t.txNode.Tombstone = marked
		t.putNode(tableKey(tableTransactionRootByWorldState, t.rootHash), t.txNode)
		t.rootHash = &hash
		t.rootNode = rootNode
//		t.logger.Info("Rebased world state to hash %x", hash)
		t.txNode = txNode
//		t.logger.Info("Rebased transaction trie hash %x", *txNode.hash())
	}
	return nil
}

func (t *MptWorldState) Cleanup(hash core.Byte64) error {
	// lookup node for the provided hash
	if staleNode, ok := t.getState(&hash); !ok {
		t.logger.Error("Failed to find stale node for hash %x", hash)
		return core.NewCoreError(ERR_NOT_FOUND, "hash does not exists")
	} else {
		// node exists, perform a depth first cleanup of world state
		t.cleanupDepthFirst(t.deleteState, t.getState, staleNode)
		t.logger.Debug("Cleaned up world state %x", hash)
		// now perform a depth first cleanup of transactions
		if txNode, _ := t.getNode(tableKey(tableTransactionRootByWorldState, &hash)); txNode != nil {
			t.cleanupDepthFirst(t.deleteTx, t.getTx, txNode)
			// also delete the old world state key for transaction trie root
			t.deleteNode(tableKey(tableTransactionRootByWorldState, &hash))
			t.logger.Debug("Cleaned up transaction trie %x", *txNode.hash())
		}
	}
	return nil
}

func (t *MptWorldState) cleanupDepthFirst(del deleter, get getter, currNode *node) {
	// check if we have reached the end of stale trie
	if currNode == nil || currNode.Tombstone == unmarked {
//		t.logger.Debug("Cleanup reached active node %x", *currNode.hash())
		return
	}
	// else continue to cleanup depth first
	for _, childHash := range currNode.Idxs {
		if childHash == null {
			continue
		}
        if childNode, found := get(&childHash); found {
		    t.cleanupDepthFirst(del, get, childNode)
        } else {
	        	t.logger.Error("Node not found for hash %x", childHash)
        }
	}
	// remove the current node from DB
	if err := del(currNode.hash()); err != nil {
		t.logger.Error("%x: %s", *currNode.hash(), err)
	} else {
//		t.logger.Debug("Deleted stale nodes for hash %x", *currNode.hash())
	}
}
