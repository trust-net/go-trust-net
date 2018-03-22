package consensus

import (
	"github.com/trust-net/go-trust-net/core"
)

// A blockchain based consensus platform implementation
type BlockChainConsensus struct {
	
}

func NewBlockChainConsensus() (*BlockChainConsensus, error) {
	c := BlockChainConsensus{}
	return &c, nil
}

// get a new "candidate" block, initialized with a copy of world state
// from request time tip of the canonical chain
func (c *BlockChainConsensus) NewCandidateBlock() Block {
	return nil
}

// submit a "filled" block for mining, and it will mine the block and update canonical chain
// (or abort if a new network block is received with same or higher weight)
func (c *BlockChainConsensus) MineCandidateBlock(b Block) error {
	return core.NewCoreError(ERR_NOT_IMPLEMENTED, "mining not yet implemented")
}

// query status of a transaction (its block details) in the canonical chain
func (c *BlockChainConsensus) TransactionStatus(tx *Transaction) (Block, error) {
	return nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "transaction status not yet implemented")
}

// deserialize data into network block, and will initialize the block with current canonical parent's
// world state root (application is responsible to run the transactions from block, and update
// world state appropriately)
func (c *BlockChainConsensus) Deserialize(data []byte) (Block, error) {
	return nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "deserialize not yet implemented")
}

// submit a "validated" network block, and it will add to block DAG appropriately
// (i.e. either extend canonical chain, or add as an uncle block), will also update
// the world state with block's transactions
func (c *BlockChainConsensus) AcceptNetworkBlock(b Block) error {
	return core.NewCoreError(ERR_NOT_IMPLEMENTED, "accept network block not yet implemented")
}

// serialize a block to send over wire
func (c *BlockChainConsensus) Serialize(b Block) ([]byte, error) {
	return nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "serialize not yet implemented")
}
