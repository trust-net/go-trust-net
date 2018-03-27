package trie

import (
	"github.com/trust-net/go-trust-net/core"
)

// an interface to implement underlying datastructure for maintaining
// world state, any update to world state would result in a new root hash,
// which is a cryptographic signature of the world state
type WorldState interface {
	// update a key with value, return root hash after change
	Update(key, value []byte) core.Byte64
//	// update a key with value, along with updating the transaction ID causing this change,
//	// return root hash after change
//	UpdateWithTx(txId *core.Byte64, key, value []byte) core.Byte64
	// delete a key, return root hash after change
	// a delete on non existing key simply does not change root hash
	Delete(key []byte) core.Byte64
//	// delete a key, along with updating the transaction ID causing this change,
//	// return root hash after change
//	// a delete on non existing key simply does not change root hash
//	DeleteWithTx(key []byte) core.Byte64
	// lookup value for a key, return value and error if not successful
	// a lookup on non existing key is considered an error
	Lookup(key []byte) ([]byte, error)
	// lookup a transaction, if exists return block hash where this transaction finalized as per world state view
	// and error if not successful (a lookup on non existing transaction returns error)
	HasTransaction(txId *core.Byte64) (*core.Byte64, error)
	// register a finalized transaction and associated block based on current world view
	// after one or more world state updates, there must be a transaction registration
	// (it is assumed, after world state updates due to a transaction, that transaction itself
	// will also get registered to tie into new world state view after updates) 
	RegisterTransaction(txId, blockId *core.Byte64) error
	// a getter for current world state cryptographic signature (i.e. root hash)
	Hash() core.Byte64
	// rebase the world state to a different root hash
	// (this will also atomically rebase transaction view to match world state view)
	// returns an error if provided root hash is not a valid existing hash
	Rebase(hash core.Byte64) error
	// cleanup the tombstoned nodes in the underlying data structure
	// from specified world state hash
	Cleanup(hash core.Byte64) error
}
