package consensus

import (
	"sync"
    "time"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/core/trie"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/common"
)

const (
	maxBlocks = 100
	maxUncleDistance = uint64(5)
)

var tableChainNode = []byte("ChainNode-")
var tableBlock = []byte("Block-")
var dagTip = []byte("ChainState-DagTip")
func tableKey(prefix []byte, key *core.Byte64) []byte {
	return append(prefix, key.Bytes()...)
}


// A blockchain based consensus platform implementation
type BlockChainConsensus struct {
	state trie.WorldState
	tip *block
	genesisNode *chainNode
	minerId *core.Byte64
	db db.Database
	lock sync.RWMutex
	logger log.Logger
}

// application is responsible to create an instance of DB initialized to application's name space
func NewBlockChainConsensus(genesisHash *core.Byte64, genesisTime uint64,
	minerId *core.Byte64, db db.Database) (*BlockChainConsensus, error) {
	chain := BlockChainConsensus{
		db: db,
		minerId: minerId,
		state: trie.NewMptWorldState(db),
	}
	chain.logger = log.NewLogger(chain)

	// genesis is statically defined using default values
	genesisBlock := newBlock(core.BytesToByte64(nil), 0, 0, genesisTime, minerId, chain.state)
	genesisBlock.hash = genesisHash
	chain.genesisNode = newChainNode(genesisBlock)
	chain.genesisNode.setMainList(true)

	// read tip hash from DB
	if data, err := chain.db.Get(dagTip); err != nil {
		chain.logger.Debug("No DAG tip in DB, using genesis as tip")
		chain.tip = genesisBlock
		// save the tip in DB
		if err := chain.db.Put(dagTip, chain.tip.Hash().Bytes()); err != nil {
			chain.logger.Error("Failed to save DAG tip in DB: %s", err.Error())
			return nil, core.NewCoreError(core.ERR_DB_UNINITIALIZED, err.Error())
		}
		// following is needed so that any immediate chid can be consistently operated on,
		// e.g. when traversing ancestors to find uncles
		// save the genesis block in DB
		if err := chain.putBlock(genesisBlock); err != nil {
			chain.logger.Error("Failed to save genesis block in DB: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
		if err := chain.putChainNode(chain.genesisNode); err != nil {
			chain.logger.Error("Failed to save genesis chain node in DB: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
	} else {
		hash := core.BytesToByte64(data)
		chain.logger.Debug("Found DAG tip from DB: '%x'", *hash)
		// read the tip block
		var err error
		if chain.tip, err = chain.getBlock(hash); err != nil {
			chain.logger.Error("Failed to get tip block from DB: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
		// rebase state trie to current tip
		if err = chain.state.Rebase(chain.tip.STATE); err != nil {
			chain.logger.Error("Failed to rebase world state to tip's world state: %s", err.Error())
			return nil, core.NewCoreError(ERR_INITIALIZATION_FAILED, err.Error())
		}
	}
	chain.logger.Debug("Initialized block chain DAG")
	return &chain, nil
}

// return the tip of current canonical blockchain
func (c *BlockChainConsensus) Tip() Block {
	return c.tip
}


func (chain *BlockChainConsensus) getChainNode(hash *core.Byte64) (*chainNode, error) {
	if data, err := chain.db.Get(tableKey(tableChainNode, hash)); err != nil {
		chain.logger.Error("Did not find chain node in DB: %s", err.Error())
		return nil, err
	} else {
		var node chainNode
		if err := common.Deserialize(data, &node); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, err
		}
		return &node, nil
	}
}

func (chain *BlockChainConsensus) getBlock(hash *core.Byte64) (*block, error) {
	if data, err := chain.db.Get(tableKey(tableBlock, hash)); err != nil {
		chain.logger.Error("Did not find block in DB: %s", err.Error())
		return nil, err
	} else {
		var block *block
		var err error
		if block, err = deSerializeBlock(data); err != nil {
			chain.logger.Error("failed to decode data from DB: %s", err.Error())
			return nil, err
		}
		return block, nil
	}
}

// persist a blocknode into DB
func (chain *BlockChainConsensus) putChainNode(node *chainNode) error {
	if data, err := common.Serialize(node); err == nil {
		return chain.db.Put(tableKey(tableChainNode, node.hash()), data)
	} else {
		return err
	}
	
}

// persist a block into DB
func (chain *BlockChainConsensus) putBlock(block *block) error {
	if data, err := serializeBlock(block); err == nil {
		return chain.db.Put(tableKey(tableBlock, block.Hash()), data)
	} else {
		return err
	}
}

// get a new "candidate" block, initialized with a copy of world state
// from request time tip of the canonical chain
func (c *BlockChainConsensus) NewCandidateBlock() Block {
	c.lock.RLock()
	defer c.lock.RUnlock()
	b := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	return b
}

// submit a "filled" block for mining (executes as a goroutine)
// it will mine the block and update canonical chain or abort if a new network block
// is received with same or higher weight, the callback MiningResultHandler will be called
// with serialized data for the block that can be  sent over the wire to peers,
// or error if mining failed/aborted
func (c *BlockChainConsensus) MineCandidateBlock(b Block, cb MiningResultHandler) {
	go c.mineCandidateBlock(b, cb)
}
func (c *BlockChainConsensus) mineCandidateBlock(b Block, cb MiningResultHandler) {
	cb(nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "mining not yet implemented"))
}

// query status of a transaction (its block details) in the canonical chain
func (c *BlockChainConsensus) TransactionStatus(tx *Transaction) (Block, error) {
	return nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "transaction status not yet implemented")
}

// validate uncle relationship
func (c *BlockChainConsensus) isUncleValid(child *block, uHash *core.Byte64) bool {
	// check if uncle is known
	if uncle, err := c.getChainNode(uHash); err != nil {
		c.logger.Error("block with unknown uncle!!!")
		return false
	} else {
		// find common ancestor
		var parent *chainNode
		for parent, _ = c.getChainNode(&child.PHASH); parent != nil && parent.Depth < uncle.Depth; {
			parent, _ = c.getChainNode(parent.Parent)
		}
		for ; parent != nil && uncle != nil && uncle.Depth < parent.Depth; {
			uncle, _ = c.getChainNode(uncle.Parent)
		}
		if parent == nil || uncle == nil {
			// did not find common ancestor
			return false
		}
		if *parent.Hash == *uncle.Hash {
			// uncle cannot be direct ancestor
			return false
		}
		for parent != nil && uncle != nil && *parent.Hash != *uncle.Hash {
			parent, _ = c.getChainNode(parent.Parent)
			uncle, _ = c.getChainNode(uncle.Parent)
		}
		return parent != nil && uncle != nil && (child.Depth().Uint64() - parent.Depth <  maxUncleDistance)
	}
}
// deserialize data into network block, and will initialize the block with block's parent's
// world state root (application is responsible to run the transactions from block, and update
// world state appropriately)
func (c *BlockChainConsensus) DeserializeNetworkBlock(data []byte) (Block, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	var block, parent *block
	var err error
	if block, err = deSerializeBlock(data); err != nil {
		c.logger.Error("failed to deserialize network block's data: %s", err.Error())
		return nil, err
	}
	// verify that block's parent exists
	if parent, err = c.getBlock(block.ParentHash()); err != nil {
		c.logger.Error("failed to find network block's parent: %s", err.Error())
		return nil, core.NewCoreError(ERR_BLOCK_ORPHAN, "cannot find parent")
	}
	// validate that block's depth is correct
	if block.Depth().Uint64() != parent.Depth().Uint64() + 1 {
		c.logger.Error("incorrect depth on network block")
		return nil, core.NewCoreError(ERR_BLOCK_VALIDATION, "incorrect depth")
	}
	// validate that block's weight is correct
	weight := parent.Weight().Uint64() + 1
	// validate uncles are known and within distance
	for _, uncle := range block.Uncles() {
		if !c.isUncleValid(block, &uncle) {
			c.logger.Error("block with invalid uncle: %x", uncle)
			return nil, core.NewCoreError(ERR_BLOCK_VALIDATION, "invalid uncle")
		}
		// increment weight for valid uncle
		weight++
		// TODO: process mining reward for uncle list
	}

	if block.Weight().Uint64() != weight {
		c.logger.Error("incorrect weight on network block")
		return nil, core.NewCoreError(ERR_BLOCK_VALIDATION, "incorrect weight")
	}
	// initialze block's world state to parent's world state
	state := trie.NewMptWorldState(c.db)
	if err = state.Rebase(parent.STATE); err != nil {
		c.logger.Error("failed to initialize network block's world state: %s", err.Error())
		return nil, core.NewCoreError(ERR_STATE_INCORRECT, "cannot initialize state")
	}
	block.worldState = state
	return block, nil
}

// submit a "processed" network block, will be added to DAG appropriately
// (i.e. either extend canonical chain, or add as an uncle block)
// block's computed world state should match STATE of the deSerialized block,
func (c *BlockChainConsensus) AcceptNetworkBlock(b Block) error {
	return core.NewCoreError(ERR_NOT_IMPLEMENTED, "accept network block not yet implemented")
}

//// serialize a block to send over wire
//func (c *BlockChainConsensus) Serialize(b Block) ([]byte, error) {
//	return nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "serialize not yet implemented")
//}
