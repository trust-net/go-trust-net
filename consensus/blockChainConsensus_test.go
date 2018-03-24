package consensus

import (
    "testing"
    "time"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core/trie"
)

var genesisHash *core.Byte64
var genesis *block
var genesisTime = uint64(0x123456)
var testNode = core.BytesToByte64([]byte("a test node"))

func init() {
	db, _ := db.NewDatabaseInMem()
	genesis := newBlock(core.BytesToByte64(nil), 0, 0, genesisTime, core.BytesToByte64(nil), trie.NewMptWorldState(db))
	genesisHash = genesis.computeHash()
}

func TestNewBlockChainConsensusGenesis(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain instance implements Consensus interface
	var consensus Consensus
	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	if consensus == nil {
		t.Errorf("got nil instance")
		return
	}
	if *consensus.(*BlockChainConsensus).Tip().Hash() != *genesisHash {
		t.Errorf("tip is not genesis:\nExpected %x\nFound %x", *genesisHash, *consensus.(*BlockChainConsensus).Tip().Hash())
	}
}

func TestNewBlockChainConsensusPutChainNode(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain reports error when cannot save dag tip 
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	err = c.putChainNode(&chainNode{Hash: core.BytesToByte64(nil),})
	if err != nil{
		t.Errorf("failed to put chain node into db: %s", err)
		return
	}
}

func TestNewBlockChainConsensusFromTip(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if c == nil {
		t.Errorf("got nil instance")
		return
	}
	// move the tip to a new block
	child := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	tip := child.computeHash()
	if err := db.Put(dagTip, tip.Bytes()); err != nil {
			t.Errorf("failed to save new DAG tip: %s", err)
	}
	if err := c.putBlock(child); err != nil {
		t.Errorf("failed to save new block: %s", err)
	}
	if err := c.putChainNode(newChainNode(child)); err != nil {
		t.Errorf("failed to save new chain node: %s", err)
	}
	
	log.SetLogLevel(log.NONE)
	// at this time, DB should have tip pointing to child block above,
	// so create a new instance of blockchain and check if it initialized correctly
	c, err = NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	if c == nil {
		t.Errorf("got nil instance")
		return
	}
	if *c.Tip().Hash() != *tip {
		t.Errorf("tip is not correct:\nExpected %x\nFound %x", *tip, *c.Tip().Hash())
	}
	if c.weight != child.Weight().Uint64() {
		t.Errorf("weight is not correct: Expected %d Found %d", child.Weight().Uint64(), c.weight)
	}
}

func TestNewCandidateBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain instance implements Consensus interface
	c, _ := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if c == nil {
		t.Errorf("got nil instance")
		return
	}
	b := c.NewCandidateBlock()
	if b == nil {
		t.Errorf("got nil candidate block")
		return
	}
	// parent of the candidate should be current tip of blockchain
	if *b.ParentHash() != *c.Tip().Hash() {
		t.Errorf("incorrect parent:\nExpected %x\nFound %x", *c.Tip().Hash(), *b.ParentHash())
	}
}

func TestMineCandidateBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// get a new candidate block
	child := c.NewCandidateBlock()
	// add a transaction to the candidate block
	child.Update([]byte("key"), []byte("value"))
	tx := testTransaction("transaction 1")
	if err := child.AddTransaction(tx); err != nil {
		t.Errorf("failed to add transaction: %s", err)
	}
	// mining will be executed in a background goroutine
	log.SetLogLevel(log.NONE)
	done := make(chan struct{})
	c.MineCandidateBlock(child, func(data []byte, err error) {
			defer func() {done <- struct{}{}}()
			if err != nil {
				t.Errorf("failed to mine candidate block: %s", err)
				return
			}
			// canonical chain's tip should match child node
			if *c.Tip().Hash() != *child.Hash() {
				t.Errorf("Canonical chain tip does not match mined block")
			}
			// world view should also match
			if c.state.Hash() != child.(*block).STATE {
				t.Errorf("World state not updated after mining")
			}
	});
	// wait for our callback to finish
	<-done
}

func TestMineCandidateBlockDuplicate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// get a new candidate block
	child := c.NewCandidateBlock()
	// mining will be executed in a background goroutine
	log.SetLogLevel(log.NONE)
	done := make(chan struct{})
	c.MineCandidateBlock(child, func(data []byte, err error) {
			defer func() {done <- struct{}{}}()
			if err != nil {
				t.Errorf("failed to mine candidate block: %s", err)
				return
			}
	});
	// wait for our callback to finish
	<-done
	// now re-submit the same block for mining
	c.MineCandidateBlock(child, func(data []byte, err error) {
			defer func() {done <- struct{}{}}()
			if err == nil {
				t.Errorf("failed to detect duplicate candidate block")
			}
	});
	// wait for our callback to finish
	<-done
}

func TestTransactionStatus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	
	// get a new candidate block
	child := c.NewCandidateBlock()
	// add a transaction to the candidate block
	tx := testTransaction("transaction 1")
	if err := child.AddTransaction(tx); err != nil {
		t.Errorf("failed to add transaction: %s", err)
	}
	// mine the child block
	done := make(chan struct{})
	c.MineCandidateBlock(child, func(data []byte, err error) {
			defer func() {done <- struct{}{}}()
			if err != nil {
				t.Errorf("failed to mine candidate block: %s", err)
				return
			}
	});
	// wait for our callback to finish
	<-done

	// now query for the transaction
	var b Block
	if b,err = c.TransactionStatus(tx); err != nil {
		t.Errorf("failed to get transaction status: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
	if *b.Hash() != *child.Hash() {
		t.Errorf("transaction has incorrect block")
	}
}

func TestDeserializeNetworkBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// build a new block to simulate current tip's child
	child := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	child.computeHash()
	data,_ := serializeBlock(child)
	var b Block
	if b, err = c.DeserializeNetworkBlock(data); err != nil {
		t.Errorf("failed to deserialize block: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
	if b.(*block).worldState.Hash() != c.tip.STATE {
		t.Errorf("Incorrect state initialization of network block:\nExpected %x\nFound %x", c.tip.STATE, b.(*block).worldState.Hash())
	}
	if !b.(*block).isNetworkBlock {
		t.Errorf("network flag not set")
		return
	}
}

func TestDeserializeNetworkBlockWithUncle(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// add an uncle block to blockchain
	uncle := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	uncle.computeHash()
	c.putBlock(uncle)
	c.putChainNode(newChainNode(uncle))

	// build a parent block to blockchain
	parent := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	parent.computeHash()
	c.putBlock(parent)
	c.putChainNode(newChainNode(parent))

	// build a new block simulating parent's child and uncle's nephew
	child := newBlock(parent.Hash(), parent.Weight().Uint64() + 1 + 1, parent.Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	child.UNCLEs = append(child.UNCLEs, *uncle.Hash())
	child.computeHash()
	data,_ := serializeBlock(child)
	var b Block
	if b, err = c.DeserializeNetworkBlock(data); err != nil {
		t.Errorf("failed to deserialize block: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
	if b.Weight().Uint64() != parent.Weight().Uint64() + 1 + 1 {
		t.Errorf("Incorrect weight initialization of network block:\nExpected %d, Found %d", parent.Weight().Uint64() + 1 + 1, b.Weight().Uint64())
	}
}

func TestAcceptNetworkBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// build a new block to simulate current tip's child
	child := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	child.computeHash()
	data,_ := serializeBlock(child)
	var b Block
	if b, err = c.DeserializeNetworkBlock(data); err != nil {
		t.Errorf("failed to deserialize block: %s", err)
		return
	}
	if err = c.AcceptNetworkBlock(b); err != nil {
		t.Errorf("failed to accept network block: %s", err)
		return
	}
	// current tip should move to this new block
	if *c.Tip().Hash() != *child.Hash() {
		t.Errorf("DAG tip did not update!")
		return
	}
	if c.state.Hash() != child.STATE {
		t.Errorf("DAG world state did not update!")
		return
	}
}

func TestAcceptNetworkBlockDuplicate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// build a new block to simulate current tip's child
	child := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	child.computeHash()
	data,_ := serializeBlock(child)
	var b Block
	if b, err = c.DeserializeNetworkBlock(data); err != nil {
		t.Errorf("failed to deserialize block: %s", err)
		return
	}
	if err = c.AcceptNetworkBlock(b); err != nil {
		t.Errorf("failed to accept network block: %s", err)
		return
	}
	// try re-submitting same block again
	if err = c.AcceptNetworkBlock(b); err == nil || err.(*core.CoreError).Code() != ERR_DUPLICATE_BLOCK {
		t.Errorf("failed to detect duplicate network block")
	}
}

func addBlock(b Block, c Consensus) error {
	// mining will be executed in a background goroutine
	log.SetLogLevel(log.NONE)
	done := make(chan struct{})
	var result error
	c.MineCandidateBlock(b, func(data []byte, err error) {
			result = err
			defer func() {done <- struct{}{}}()
//			if err != nil {
//				t.Errorf("failed to mine candidate block: %s", err)
//				return
//			}
	});
	// wait for our callback to finish
	<-done
	return result
}


func makeBlocks(len int, parent *block, c *BlockChainConsensus) []Block {
	nodes := make([]Block, len)
	for i := uint64(0); i < uint64(len); i++ {
		state := trie.NewMptWorldState(c.db)
		state.Rebase(c.state.Hash())
		child := newBlock(parent.Hash(), parent.Weight().Uint64()+1, parent.Depth().Uint64()+1, 0, testMiner, state)
		child.computeHash()
		nodes[i] = child
		parent = child
	}
	return nodes
}

func addChain(chain *BlockChainConsensus, blocks []Block) error{
//	var parent *block
//	parent = nil  
	for _, block := range(blocks) {
//		if parent != nil {
//			block = core.NewSimpleBlock(parent.Hash(), parent.Weight().Uint64()+1, parent.Depth().Uint64()+1, 0, miner)
//		}
		if err := chain.AcceptNetworkBlock(block); err != nil {
			return err
		}
//		parent = block
//		blocks[i] = block
	}
	return nil
}


func TestBlockChainConsensusHeaviestChain(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// add an ancestor block to chain
	ancestor := c.NewCandidateBlock()
	if err := addBlock(ancestor, c); err != nil {
		t.Errorf("failed to add block: %s", err)
	}

	// now add 1st chain with 6 blocks after the ancestor
	
	chain1 := makeBlocks(6, ancestor.(*block), c)
	if err := addChain(c, chain1); err != nil {
		t.Errorf("1st chain failed to add block: %s", err)
	}
	// now add 2nd chain with 4 blocks after the ancestor	
	chain2 := makeBlocks(4, ancestor.(*block), c)
	log.SetLogLevel(log.DEBUG)
	if err := addChain(c, chain2); err != nil {
		t.Errorf("2nd chain failed to add block: %s", err)
	}
	log.SetLogLevel(log.NONE)
	// validate that heaviest chain (chain1, length 1+6) wins
	if c.Tip().Depth().Uint64() != 7 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 7, c.Tip().Depth().Uint64())
	}
	if c.weight != chain1[5].Weight().Uint64() {
		t.Errorf("chain weight incorrect: Expected '%d' Found '%d'", chain1[5].Weight().Uint64(), c.weight)
	}
}
