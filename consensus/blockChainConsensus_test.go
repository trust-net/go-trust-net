package consensus

import (
    "testing"
    "time"
    "fmt"
    "math/rand"
	"sync"
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
//	c.MineCandidateBlock(child, func(data []byte, err error) {
	c.MineCandidateBlock(child, func(b Block, err error) {
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
//	c.MineCandidateBlock(child, func(data []byte, err error) {
	c.MineCandidateBlock(child, func(block Block, err error) {
			defer func() {done <- struct{}{}}()
			if err != nil {
				t.Errorf("failed to mine candidate block: %s", err)
				return
			}
	});
	// wait for our callback to finish
	<-done
	// now re-submit the same block for mining
//	c.MineCandidateBlock(child, func(data []byte, err error) {
	c.MineCandidateBlock(child, func(block Block, err error) {
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
//	c.MineCandidateBlock(child, func(data []byte, err error) {
	c.MineCandidateBlock(child, func(block Block, err error) {
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
	done := make(chan struct{})
	var result error
//	c.MineCandidateBlock(b, func(data []byte, err error) {
	c.MineCandidateBlock(b, func(block Block, err error) {
			result = err
			defer func() {done <- struct{}{}}()
	});
	// wait for our callback to finish
	<-done
	return result
}


func makeBlocks(len int, parent *block, c *BlockChainConsensus) []Block {
	nodes := make([]Block, len)
	for i := uint64(0); i < uint64(len); i++ {
		state := trie.NewMptWorldState(c.db)
		state.Rebase(parent.worldState.Hash())
		child := newBlock(parent.Hash(), parent.Weight().Uint64()+1, parent.Depth().Uint64()+1, 0, testMiner, state)
		child.computeHash()
		nodes[i] = child
		parent = child
	}
	return nodes
}

func addChain(chain *BlockChainConsensus, blocks []Block) error{
	for _, block := range(blocks) {
		if err := chain.AcceptNetworkBlock(block); err != nil {
			return err
		}
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
	log.SetLogLevel(log.NONE)
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

func TestBlockChainConsensusUncleWeight(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	// simulate 3 different concurrent nodes updating their individual blockchain instances
	node1, node2, node3 := core.BytesToByte64([]byte("test node #1")), core.BytesToByte64([]byte("test node #2")), core.BytesToByte64([]byte("test node #3"))
	db1, _ := db.NewDatabaseInMem()
	chain1, _ := NewBlockChainConsensus(genesisHash, genesisTime, node1, db1)
	db2, _ := db.NewDatabaseInMem()
	chain2, _ := NewBlockChainConsensus(genesisHash, genesisTime, node2, db2)
	db3, _ := db.NewDatabaseInMem()
	chain3, _ := NewBlockChainConsensus(genesisHash, genesisTime, node3, db3)
	
	// let first node mine a block and broadcast to others
	candidate1 := chain1.NewCandidateBlock()
	if err := addBlock(candidate1, chain1); err != nil {
		t.Errorf("failed to mine block: %s", err)
	}
	if err := addChain(chain2, []Block{candidate1}); err != nil {
		t.Errorf("failed to add network block: %s", err)
	}
	if err := addChain(chain3, []Block{candidate1}); err != nil {
		t.Errorf("failed to add network block: %s", err)
	}

	// now let chain2 and chain3 mine blocks in parallel and announce simultaneously
	candidate2 := chain2.NewCandidateBlock()
	if err := addBlock(candidate2, chain2); err != nil {
		t.Errorf("failed to mine block: %s", err)
	}
	candidate3 := chain3.NewCandidateBlock()
	if err := addBlock(candidate3, chain3); err != nil {
		t.Errorf("failed to mine block: %s", err)
	}
	if err := addChain(chain1, []Block{candidate2}); err != nil {
		t.Errorf("failed to add network block: %s", err)
	}
	if err := addChain(chain3, []Block{candidate2}); err != nil {
		t.Errorf("failed to add network block: %s", err)
	}
	if err := addChain(chain1, []Block{candidate3}); err != nil {
		t.Errorf("failed to add network block: %s", err)
	}
	if err := addChain(chain2, []Block{candidate3}); err != nil {
		t.Errorf("failed to add network block: %s", err)
	}
	// now, next candidate block on chain1 should have candidate2 (first recieved) as parent,
	// and candidate3 (next recieved) as uncle
	log.SetLogLevel(log.DEBUG)
	candidate1 = chain1.NewCandidateBlock()
	if *candidate1.ParentHash() != *candidate2.Hash() {
		t.Errorf("incorrect parent hash")
	}
	if len(candidate1.Uncles()) != 1 || candidate1.Uncles()[0] != *candidate3.Hash() {
		t.Errorf("incorrect uncles: %d, %x", len(candidate1.Uncles()),  candidate1.Uncles()[0])
	}	
}

func TestBlockChainConsensus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	// simulate 3 different concurrent nodes updating their individual blockchain instances
	node1, node2, node3 := core.BytesToByte64([]byte("test node #1")), core.BytesToByte64([]byte("test node #2")), core.BytesToByte64([]byte("test node #3"))
	db1, _ := db.NewDatabaseInMem()
	chain1, _ := NewBlockChainConsensus(genesisHash, genesisTime, node1, db1)
	db2, _ := db.NewDatabaseInMem()
	chain2, _ := NewBlockChainConsensus(genesisHash, genesisTime, node2, db2)
	db3, _ := db.NewDatabaseInMem()
	chain3, _ := NewBlockChainConsensus(genesisHash, genesisTime, node3, db3)

	// define an application for this consensus platform
	counter := 0
	lock := sync.RWMutex{}
	nodeFunc := func(myChain *BlockChainConsensus, myNode *core.Byte64) {
		// wait random time and grab lock for one instance of application
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(200)))
		lock.Lock()
		defer lock.Unlock()

		// create a new candidate block
		candidate := myChain.NewCandidateBlock()

		// create a mining callback handler for this candidate block
		done := make(chan struct{})
		var minedBlock []byte
//		miningCallback := func(data []byte, err error) {
		miningCallback := func(block Block, err error) {
				defer func() {
					done <- struct{}{}
				}()
				if err != nil {
					t.Errorf("failed to mine candidate block: %s", err)
					minedBlock = nil
					return
				}
				// simulate mining delay
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(200)))
				minedBlock, err = serializeBlock(block)
		}
		// process mining block for application instance, and then broadcast to network
		switch myChain {
			case chain1:
				// add local block
				chain1.MineCandidateBlock(candidate, miningCallback)
				// wait for our callback to finish
				<-done
				// broadcast network block
				if b, err := chain2.DeserializeNetworkBlock(minedBlock); err != nil {
					t.Errorf("failed to deserialize block: %s", err)
				} else {
					if err := chain2.AcceptNetworkBlock(b); err != nil {
						t.Errorf("failed to accept block: %s", err)
					}
				}
				if b, err := chain3.DeserializeNetworkBlock(minedBlock); err != nil {
					t.Errorf("failed to deserialize block: %s", err)
				} else {
					if err := chain3.AcceptNetworkBlock(b); err != nil {
						t.Errorf("failed to accept block: %s", err)
					}
				}
			case chain2:
				// add local block
				chain2.MineCandidateBlock(candidate, miningCallback)
				// wait for our callback to finish
				<-done
				// broadcast network block
				if b, err := chain1.DeserializeNetworkBlock(minedBlock); err != nil {
					t.Errorf("failed to deserialize block: %s", err)
				} else {
					if err := chain1.AcceptNetworkBlock(b); err != nil {
						t.Errorf("failed to accept block: %s", err)
					}
				}
				if b, err := chain3.DeserializeNetworkBlock(minedBlock); err != nil {
					t.Errorf("failed to deserialize block: %s", err)
				} else {
					if err := chain3.AcceptNetworkBlock(b); err != nil {
						t.Errorf("failed to accept block: %s", err)
					}
				}
			case chain3:
				// add local block
				chain3.MineCandidateBlock(candidate, miningCallback)
				// wait for our callback to finish
				<-done
				// broadcast network block
				if b, err := chain1.DeserializeNetworkBlock(minedBlock); err != nil {
					t.Errorf("failed to deserialize block: %s", err)
				} else {
					if err := chain1.AcceptNetworkBlock(b); err != nil {
						t.Errorf("failed to accept block: %s", err)
					}
				}
				if b, err := chain2.DeserializeNetworkBlock(minedBlock); err != nil {
					t.Errorf("failed to deserialize block: %s", err)
				} else {
					if err := chain2.AcceptNetworkBlock(b); err != nil {
						t.Errorf("failed to accept block: %s", err)
					}
				}
		}
		counter++
		fmt.Printf("%s : chain depth: %d, chain weight: %d, Counter: %d\n", *myNode, myChain.Tip().Depth().Uint64(), myChain.Tip().Weight().Uint64(), counter)
	}
	
	// run the node functions on 3 nodes concurrently
	for i := 0; i < 10; i++ {
		go nodeFunc(chain1, node1)
		go nodeFunc(chain2, node2)
		go nodeFunc(chain3, node3)
	}
	// wait for all nodes to finish
	for counter < 30 {time.Sleep(time.Millisecond * 100)}
	// validate that all 3 chains have same tip node hash
	if *chain1.Tip().Hash() != *chain2.Tip().Hash() {
		t.Errorf("tip of chain1 and chain2 are different")
	}
	if *chain2.Tip().Hash() != *chain3.Tip().Hash() {
		t.Errorf("tip of chain2 and chain3 are different")
	}
	// validate that all 3 chains have same depth of main/longest chain
	if *chain1.Tip().Depth() != *chain2.Tip().Depth() {
		t.Errorf("Depth of chain1 '%d' not same as chain2 '%d'", chain1.Tip().Depth().Uint64(), chain2.Tip().Depth().Uint64())
	}
	if *chain2.Tip().Depth() != *chain3.Tip().Depth() {
		t.Errorf("Depth of chain2 '%d' not same as chain3 '%d'", chain2.Tip().Depth().Uint64(), chain3.Tip().Depth().Uint64())
	}
	// validate that all 3 chains have same TD
	if *chain1.Tip().Weight() != *chain2.Tip().Weight() {
		t.Errorf("TD of chain1 '%d' not same as chain2 '%d'", chain1.Tip().Weight().Uint64(), chain2.Tip().Weight().Uint64())
	}
	if chain2.Tip().Weight().Uint64() != chain3.Tip().Weight().Uint64() {
		t.Errorf("TD of chain2 '%d' not same as chain3 '%d'", chain2.Tip().Weight().Uint64(), chain3.Tip().Weight().Uint64())
	}
}

func TestBlockChainConsensusBestBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}

	// add few blocks to chain
	if err := addChain(c, makeBlocks(3, c.tip, c)); err != nil {
		t.Errorf("failed to add block: %s", err)
	}
	
	// get the best block
	bb := c.BestBlock()
	if bb == nil {
		t.Errorf("failed to get best block")
		return
	}
	// validate the best block
	if *bb.Hash() != *c.tip.Hash() {
		t.Errorf("best block hash incorrect: %x", *bb.Hash())
	}
	if *bb.Depth() != *c.tip.Depth() {
		t.Errorf("best block Depth incorrect: %d", bb.Depth().Uint64())
	}
	if *bb.Weight() != *c.tip.Weight() {
		t.Errorf("best block Weight incorrect: %d", bb.Weight().Uint64())
	}
}

func TestBlockChainConsensusDescendents(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
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
	// re assigned mined block to ancestor
	ancestor = c.BestBlock()

	// add few blocks to chain
	if err := addChain(c, makeBlocks(3, ancestor.(*block), c)); err != nil {
		t.Errorf("failed to add block: %s", err)
	}
	
	// fetch descendents from ancestor
	if descendents, err := c.Descendents(ancestor.Hash(), 100); err != nil {
		t.Errorf("failed to get descendents: %s", err)
	} else {
		if len(descendents) != 3 {
			t.Errorf("did not get all descendents: %d", len(descendents))
		}
		// validate each descendent
		for _, descendent := range descendents {
			if _, err := c.DeserializeNetworkBlock(descendent); err != nil {
				t.Errorf("failed to de-serialize descendent: %s", err)
			}
		}
	}
}


func TestBlockChainConsensusDescendentsMaxBlocks(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
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
	// re assigned mined block to ancestor
	ancestor = c.BestBlock()

	// add few blocks to chain
	if err := addChain(c, makeBlocks(100, ancestor.(*block), c)); err != nil {
		t.Errorf("failed to add block: %s", err)
	}
	
	// fetch descendents from ancestor
	if descendents, err := c.Descendents(ancestor.Hash(), 10); err != nil {
		t.Errorf("failed to get descendents: %s", err)
	} else {
		if len(descendents) != 10 {
			t.Errorf("did not limit descendents to max requested size: %d", len(descendents))
		}
	}
}


func TestBlockChainConsensusDescendentsMaxBlocksSystem(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
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
	// re assigned mined block to ancestor
	ancestor = c.BestBlock()

	// add few blocks to chain
	if err := addChain(c, makeBlocks(150, ancestor.(*block), c)); err != nil {
		t.Errorf("failed to add block: %s", err)
	}
	
	// fetch descendents from ancestor
	if descendents, err := c.Descendents(ancestor.Hash(), 150); err != nil {
		t.Errorf("failed to get descendents: %s", err)
	} else {
		if len(descendents) != 100 {
			t.Errorf("did not limit descendents to max system limit: %d", len(descendents))
		}
	}
}
