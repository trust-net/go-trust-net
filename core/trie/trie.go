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
	// delete a key, return root hash after change
	// a delete on non existing key simply does not change root hash
	Delete(key []byte) core.Byte64
	// lookup value for a key, return value and error if not successful
	// a lookup on non existing key is considered an error
	Lookup(key []byte) ([]byte, error)
	// a getter for current world state cryptographic signature (i.e. root hash)
	Hash() core.Byte64
	// rebase the world state to a different root hash
	// returns an error if provided root hash is not a valid existing hash
	Rebase(hash core.Byte64) error
	// cleanup the tombstoned nodes in the underlying data structure
	// from specified world state hash
	Cleanup(hash core.Byte64) error
}
