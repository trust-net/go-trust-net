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

// following is an implementation of DB interface that can be mocked out to send specific error responses
type errorDb struct {
	step int
	values []error
}
func (db *errorDb) Put(key []byte, value []byte) error {
	db.step++
	return db.values[db.step-1]
}

func (db *errorDb) Get(key []byte) ([]byte, error) {
	db.step++
	return nil, db.values[db.step-1]
}

func (db *errorDb) Has(key []byte) (bool, error) {
	db.step++
	return false, db.values[db.step-1]
}

func (db *errorDb) Delete(key []byte) error {
	db.step++
	return db.values[db.step-1]
}

func (db *errorDb) Close() error{
	db.step++
	return db.values[db.step-1]
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestNewBlockChainConsensusErrorDagTipSave(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db := &errorDb {
		values: []error{
			nil, // trie op
			nil, // trie op
			&testError{"get dag tip error"},
			&testError{"put dag tip error"},
		},
	}
	// verify that blockchain reports error when cannot save dag tip 
	_, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err == nil || err.Error() != "put dag tip error" {
		t.Errorf("failed to report error when cannot save DAG tip: %s", err)
		return
	}
}

func TestNewBlockChainConsensusErrorGenesisBlockSave(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db := &errorDb {
		values: []error{
			nil, // trie op
			nil, // trie op
			&testError{"get dag tip error"},
			nil, // put dag tip
			&testError{"put genesis block error"},
		},
	}
	// verify that blockchain reports error when cannot save dag tip 
	_, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err == nil || err.Error() != "put genesis block error" {
		t.Errorf("failed to report error when cannot save genesis block: %s", err)
		return
	}
}


func TestNewBlockChainConsensusErrorGenesisChainNodeSave(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db := &errorDb {
		values: []error{
			nil, // trie op
			nil, // trie op
			&testError{"get dag tip error"},
			nil, // put dag tip
			nil, // put genesis block
			&testError{"put genesis chain node error"},
		},
	}
	// verify that blockchain reports error when cannot save dag tip 
	_, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err == nil || err.Error() != "put genesis chain node error" {
		t.Errorf("failed to report error when cannot save genesis block: %s", err)
		return
	}
}

func TestNewBlockChainConsensusErrorGetTipBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db := &errorDb {
		values: []error{
			nil, // trie op
			nil, // trie op
			nil, // get dag tip
			&testError{"get dag block error"},
		},
	}
	// verify that blockchain reports error when cannot save dag tip 
	_, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err == nil || err.Error() != "get dag block error" {
		t.Errorf("failed to report error when cannot get DAG block: %s", err)
		return
	}
}

func TestNewBlockChainConsensusErrorGetBlockDbError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain reports error when cannot get block 
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	// override chain db to a mock and return error
	c.db = &errorDb {
		values: []error{
			&testError{"get block error"},
		},
	}
	_, err = c.getBlock(core.BytesToByte64(nil))
	if err == nil || err.Error() != "get block error" {
		t.Errorf("failed to report error when cannot get block from db: %s", err)
		return
	}
}

func TestNewBlockChainConsensusErrorGetBlockDeSerializeError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain reports error when cannot deserialize block 
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	// override chain db to a mock and return error
	c.db = &errorDb {
		values: []error{
			nil, // get block with nil data
		},
	}
	_, err = c.getBlock(core.BytesToByte64(nil))
	if err == nil || err.Error() != "EOF" {
		t.Errorf("failed to report error when cannot deserialize block from db: %s", err)
		return
	}
}

func TestNewBlockChainConsensusErrorPutBlockError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain reports error when cannot save block 
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	// override chain db to a mock and return error
	block := newBlock(c.Tip().Hash(), c.Tip().Weight().Uint64() + 1, c.Tip().Depth().Uint64() + 1, uint64(time.Now().UnixNano()), c.minerId, c.state)
	err = c.putBlock(block)
	if err == nil || err.(*core.CoreError).Code() != ERR_BLOCK_UNHASHED {
		t.Errorf("failed to report error when cannot put block into db: %s", err)
		return
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

func TestNewBlockChainConsensusErrorGetChainNodeDbError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain reports error when cannot save dag tip 
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	// override chain db to a mock and return error
	c.db = &errorDb {
		values: []error{
			&testError{"get chain node error"},
		},
	}
	_, err = c.getChainNode(core.BytesToByte64(nil))
	if err == nil || err.Error() != "get chain node error" {
		t.Errorf("failed to report error when cannot get chain node from db: %s", err)
		return
	}
}

func TestNewBlockChainConsensusErrorGetChainNodeDeSerializeError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// verify that blockchain reports error when cannot save dag tip 
	c, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	// override chain db to a mock and return error
	c.db = &errorDb {
		values: []error{
			nil, // get block with nil data
		},
	}
	_, err = c.getChainNode(core.BytesToByte64(nil))
	if err == nil || err.Error() != "EOF" {
		t.Errorf("failed to report error when cannot deserialize chain node from db: %s", err)
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

func TestMineCandidateBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	// mining will be executed in a background goroutine
	done := make(chan struct{})
	consensus.MineCandidateBlock(nil, func(data []byte, err error) {
			defer func() {done <- struct{}{}}()
			if err != nil {
				t.Errorf("failed to mine candidate block: %s", err)
				return
			}
	});
	// wait for our callback to finish
	<-done
}

func TestTransactionStatus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	var b Block
	if b,err = consensus.TransactionStatus(nil); err != nil {
		t.Errorf("failed to get transaction status: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
}

func TestDeserializeNetworkBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	var b Block
	if b, err = consensus.DeserializeNetworkBlock(nil); err != nil {
		t.Errorf("failed to deserialize block: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
}

func TestAcceptNetworkBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	consensus, err := NewBlockChainConsensus(genesisHash, genesisTime, testNode, db)
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	if err = consensus.AcceptNetworkBlock(nil); err != nil {
		t.Errorf("failed to get accept network block: %s", err)
		return
	}
}
