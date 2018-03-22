package consensus

import (

)

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
	// submit a "validated" network block, and it will add to block DAG appropriately
	// (i.e. either extend canonical chain, or add as an uncle block), will also update
	// the world state with block's transactions
	AcceptNetworkBlock(b Block) error
//	// serialize a block to send over wire
//	Serialize(b Block) ([]byte, error)
}