package consensus

import (

)

// a consensus platform interface
type Consensus interface {
	// get a new "candidate" block, initialized with a copy of world state
	// from request time tip of the canonical chain
	NewCandidateBlock() Block
	// submit a "filled" block for mining, and it will mine the block and update canonical chain
	// (or abort if a new network block is received with same or higher weight)
	MineCandidateBlock(b Block) error
	// query status of a transaction (its block details) in the canonical chain
	TransactionStatus(tx *Transaction) (Block, error)
	// deserialize data into network block, and will initialize the block with current canonical parent's
	// world state root (application is responsible to run the transactions from block, and update
	// world state appropriately)
	Deserialize(data []byte) (Block, error)
	// submit a "validated" network block, and it will add to block DAG appropriately
	// (i.e. either extend canonical chain, or add as an uncle block), will also update
	// the world state with block's transactions
	AcceptNetworkBlock(b Block) error
	// serialize a block to send over wire
	Serialize(b Block) ([]byte, error)
}