package consensus

import (
	"github.com/trust-net/go-trust-net/core"
)

// a callback provided by application to handle results of block mining request
// serialized block is provided, if successful, or error is provided if failed/aborted
type MiningResultHandler func(data []byte, err error)

// a consensus platform interface
type Consensus interface {
	// get a new "candidate" block, initialized with a copy of world state
	// from request time tip of the canonical chain
	NewCandidateBlock() Block
	// submit a "filled" block for mining (executes as a goroutine)
	// it will mine the block and update canonical chain or abort if a new network block
	// is received with same or higher weight, the callback MiningResultHandler will be called
	// with serialized data for the block that can be  sent over the wire to peers,
	// or error if mining failed/aborted
	MineCandidateBlock(b Block, cb MiningResultHandler)
	// query status of a transaction (its block details) in the canonical chain
	TransactionStatus(tx *Transaction) (Block, error)
	// deserialize data into network block, and will initialize the block with current canonical parent's
	// world state root (application is responsible to run the transactions from block, and update
	// world state appropriately)
	DeserializeNetworkBlock(data []byte) (Block, error)
	// submit a "processed" network block, will be added to DAG appropriately
	// (i.e. either extend canonical chain, or add as an uncle block)
	// block's computed world state should match STATE of the deSerialized block,
	AcceptNetworkBlock(b Block) error
    // a copy of best block in current cannonical chain, used by protocol manager for handshake
    BestBlock() Block
    // ordered list of serialized descendents from specific parent, on the current canonical chain
    Descendents(parent *core.Byte64, max int) ([][]byte, error)
}