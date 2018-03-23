package consensus

import (
    "testing"
    "time"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/log"
)

var genesisHash = core.BytesToByte64([]byte("a test genesis hash"))
var genesisTime = uint64(0x123456)
var testNode = core.BytesToByte64([]byte("a test node"))

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

//func TestMineCandidateBlock(t *testing.T) {
//	log.SetLogLevel(log.NONE)
//	db, _ := db.NewDatabaseInMem()
//	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
//	if err != nil || consensus == nil {
//		t.Errorf("failed to get blockchain consensus instance: %s", err)
//		return
//	}
//	// mining will be executed in a background goroutine
//	done := make(chan struct{})
//	consensus.MineCandidateBlock(nil, func(data []byte, err error) {
//			defer func() {done <- struct{}{}}()
//			if err != nil {
//				t.Errorf("failed to mine candidate block: %s", err)
//				return
//			}
//	});
//	// wait for our callback to finish
//	<-done
//}

//func TestTransactionStatus(t *testing.T) {
//	log.SetLogLevel(log.NONE)
//	db, _ := db.NewDatabaseInMem()
//	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
//	if err != nil || consensus == nil {
//		t.Errorf("failed to get blockchain consensus instance: %s", err)
//		return
//	}
//	var b Block
//	if b,err = consensus.TransactionStatus(nil); err != nil {
//		t.Errorf("failed to get transaction status: %s", err)
//		return
//	}
//	if b == nil {
//		t.Errorf("got nil instance")
//		return
//	}
//}

func TestDeserializeNetworkBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || c == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// build a new block simulates current tip's child
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

//func TestAcceptNetworkBlock(t *testing.T) {
//	log.SetLogLevel(log.NONE)
//	db, _ := db.NewDatabaseInMem()
//	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
//	if err != nil || consensus == nil {
//		t.Errorf("failed to get blockchain consensus instance: %s", err)
//		return
//	}
//	if err = consensus.AcceptNetworkBlock(nil); err != nil {
//		t.Errorf("failed to get accept network block: %s", err)
//		return
//	}
//}
