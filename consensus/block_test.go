package consensus

import (
    "testing"
    "time"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/core/trie"
)

var previous = core.BytesToByte64([]byte("previous"))
var testMiner = core.BytesToByte64([]byte("some random miner"))
var testUncle = core.BytesToByte64([]byte("some random uncle"))
var testSubmitter = core.BytesToByte64([]byte("some random submitter"))

func testTransaction(payload string) *Transaction {
	return &Transaction{
		Payload: []byte(payload),
		Timestamp: core.Uint64ToByte8(uint64(time.Now().UnixNano())),
		Submitter: testSubmitter,
	}
}

func TestNewBlockWithTime(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := newBlock(previous, weight, depth, now, testMiner, ws)
	if b.Timestamp().Uint64() != now {
		t.Errorf("Block time stamp incorrect: Expected '%d', Found '%d'", now, b.Timestamp().Uint64())
	}
}

func TestNewBlockWithoutTime(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	before := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	after := uint64(time.Now().UnixNano())
	if b.Timestamp().Uint64() < before {
		t.Errorf("Block time stamp incorrect: Expected > '%d', Found '%d'", before, b.Timestamp().Uint64())
	}
	if b.Timestamp().Uint64() > after {
		t.Errorf("Block time stamp incorrect: Expected < '%d', Found '%d'", after, b.Timestamp().Uint64())
	}
}

func TestBlockStateUpdate(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	b := newBlock(previous, 0, 0, 0, testMiner, ws)
	// update some values in world state
	b.Update([]byte("key"), []byte("value1"))
	if value, err := b.Lookup([]byte("key")); err != nil {
		t.Errorf("Failed to lookup world state: %s", err)
	} else if string(value) != "value1" {
		t.Errorf("Incorrect lookup from world state: Expected '%s', Found '%s'", "value1", value)
	}
}

func TestBlockStateDelete(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	b := newBlock(previous, 0, 0, 0, testMiner, ws)
	// update some values in world state
	b.Update([]byte("key1"), []byte("value1"))
	b.Update([]byte("key2"), []byte("value2"))
	// now delete first key
	b.Delete([]byte("key1"))
	// lookup for second key should succeed
	if value, err := b.Lookup([]byte("key2")); err != nil {
		t.Errorf("Failed to lookup world state: %s", err)
	} else if string(value) != "value2" {
		t.Errorf("Incorrect lookup from world state: Expected '%s', Found '%s'", "value2", value)
	}
	// lookup of first key should fail
	if _, err := b.Lookup([]byte("key1")); err == nil {
		t.Errorf("Did not expect lookup after delete in world state")
	}
}

func TestNewBlockDefaultInitializations(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	b := newBlock(previous, 0, 0, uint64(0), testMiner, ws)
	if len(b.Transactions()) != 0 {
		t.Errorf("Block transactions list not empty: Found '%d'", len(b.Transactions()))
	}
	if len(b.Uncles()) != 0 {
		t.Errorf("Block uncles list not empty: Found '%d'", len(b.Uncles()))
	}
	if b.Hash() != nil {
		t.Errorf("Block hash not empty: Found '%x'", b.Hash())
	}
	if b.Nonce().Uint64() != 0 {
		t.Errorf("Block nonce not zero: Found '%d'", b.Nonce().Uint64())
	}
	if b.worldState.Hash() != ws.Hash() {
		t.Errorf("Block worldstate not correct:\nExpected '%x\nFound '%x'", ws.Hash(), b.worldState.Hash())
	}
}

func TestSerializeDeserializeOnTheWire(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	orig := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	orig.hash = core.BytesToByte64([]byte("some random hash"))
	orig.STATE = *core.BytesToByte64([]byte("some random state"))
	var copy block
	if data, err := common.Serialize(orig); err != nil {
		t.Errorf("Failed to serialize block: %s", err)
		return
	} else {
		if err := common.Deserialize(data, &copy); err != nil {
			t.Errorf("failed to deserialize: %s", err.Error())
			return
		}
	}
	if copy.STATE != orig.STATE {
		t.Errorf("State incorrect after deserialization: Expected '%d', Found '%d'", orig.STATE, copy.STATE)
	}
	if copy.Timestamp().Uint64() != orig.Timestamp().Uint64() {
		t.Errorf("Timestamp incorrect after deserialization: Expected '%d', Found '%d'", orig.Timestamp().Uint64(), copy.Timestamp().Uint64())
	}
	// hash should not be serialized/deserialized
	if copy.hash != nil {
		t.Errorf("Block hash got copied: Expected '%x', Found '%x'", nil, *copy.hash)
	}
	// worldstate trie should not be serialized/deserialized
	if copy.worldState != nil {
		t.Errorf("Block worldState got copied: Expected '%x', Found '%x'", nil, copy.worldState.Hash())
	}
}

func TestComputeHash(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	dbPre, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(dbPre)
	ws.Update([]byte("key"), []byte("value"))
	orig := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	tx1 := testTransaction("transaction 1")
	tx2 := testTransaction("transaction 2")
	orig.AddTransaction(tx1)
	orig.AddTransaction(tx2)
	orig.addUncle(testUncle)
	if hash := orig.computeHash(); hash == nil {
		t.Errorf("Failed to compute hash")
	}
	if hash, err := ws.HasTransaction(tx1.Id()); err != nil {
		t.Errorf("Compute hash did not update world state with transaction: %s", err)
	} else if *hash != *orig.Hash() {
		t.Errorf("Compute hash used incorrect block for transaction:\nExpected %x\nFound %x", *orig.Hash(), *hash)
	}
}

func TestSerializeDeserialize(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	dbPre, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(dbPre)
	ws.Update([]byte("key"), []byte("value"))
	orig := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	tx1 := testTransaction("transaction 1")
	tx2 := testTransaction("transaction 2")
	orig.AddTransaction(tx1)
	orig.AddTransaction(tx2)
	orig.addUncle(testUncle)
	orig.computeHash()
	var copy Block
	if data, err := serializeBlock(orig); err != nil {
		t.Errorf("Failed to serialize block: %s", err)
		return
	} else {
		if copy, err = deSerializeBlock(data); err != nil {
			t.Errorf("failed to deserialize: %s", err.Error())
			return
		}
	}
	if copy.(*block).STATE != orig.STATE {
		t.Errorf("State incorrect after deserialization: Expected '%d', Found '%d'", orig.STATE, copy.(*block).STATE)
	}
	if copy.Timestamp().Uint64() != orig.Timestamp().Uint64() {
		t.Errorf("Timestamp incorrect after deserialization: Expected '%d', Found '%d'", orig.Timestamp().Uint64(), copy.Timestamp().Uint64())
	}
	if *copy.Hash() != *orig.Hash() {
		t.Errorf("Block hash incorrect after deserialization: Expected '%x', Found '%x'", orig.Hash(), copy.Hash())
	}
	if *copy.ParentHash() != *orig.ParentHash() {
		t.Errorf("Block's parent hash incorrect after deserialization: Expected '%x', Found '%x'", orig.ParentHash(), copy.ParentHash())
	}
	if *copy.Miner() != *orig.Miner() {
		t.Errorf("Block's miner incorrect after deserialization: Expected '%x', Found '%x'", orig.Miner(), copy.Miner())
	}
	if copy.Depth().Uint64() != orig.Depth().Uint64() {
		t.Errorf("Depth incorrect after deserialization: Expected '%d', Found '%d'", orig.Depth().Uint64(), copy.Depth().Uint64())
	}
	if copy.Weight().Uint64() != orig.Weight().Uint64() {
		t.Errorf("Weight incorrect after deserialization: Expected '%d', Found '%d'", orig.Weight().Uint64(), copy.Weight().Uint64())
	}
}


func TestSerializeDeserializeWorldStateComparison(t *testing.T) {
	// prepare a world state to simulate root of parent block
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value1"))
	tx1 := testTransaction("transaction 1")
	ws.RegisterTransaction(tx1.Id(), previous)
	previousStateRoot := ws.Hash()
	
	// now build a new block starting from previous world state
	weight, depth := uint64(23), uint64(20)	
	orig := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	orig.Update([]byte("key"), []byte("value2"))
	tx2 := testTransaction("transaction 2")
	orig.AddTransaction(tx2)
	orig.addUncle(testUncle)
	orig.computeHash()
	
	// serialize and deserialize the block
	var copy Block
	if data, err := serializeBlock(orig); err != nil {
		t.Errorf("Failed to serialize block: %s", err)
		return
	} else {
		if copy, err = deSerializeBlock(data); err != nil {
			t.Errorf("failed to deserialize: %s", err.Error())
			return
		}
	}

	// now simulate re-running the transactions for this block from it's parent's world state
	copy.(*block).worldState = trie.NewMptWorldState(db)
	if err := copy.(*block).worldState.Rebase(previousStateRoot); err != nil {
		t.Errorf("failed to rebase: %s", err.Error())
		return
	}
	// key should have original value1
	if value, err := copy.(*block).worldState.Lookup([]byte("key")); err != nil || string(value) != "value1" {
		t.Errorf("unexpected value for key: Expected 'value1', Found '%s'", value)
		return
	}
	// re-play the transaction
	copy.Update([]byte("key"), []byte("value2"))
	if err := copy.AddTransaction(&copy.Transactions()[0]); err != nil {
		t.Errorf("failed to add transaction to deserialized block: %s", err.Error())
		return
	}
	// now compare world state fingerprints
	if copy.(*block).worldState.Hash() != orig.STATE {
		t.Errorf("World state fingerprint does not match: Expected '%x', Found '%x'", orig.STATE, copy.(*block).worldState.Hash())
	}
}

type invalidType struct {}

func (b *invalidType) ParentHash() *core.Byte64 {return nil}

func (b *invalidType) Miner() *core.Byte64 {return nil}

func (b *invalidType) Nonce() *core.Byte8 {return nil}

func (b *invalidType) Timestamp() *core.Byte8 {return nil}

func (b *invalidType) Depth() *core.Byte8 {return nil}

func (b *invalidType) Weight() *core.Byte8 {return nil}

func (b *invalidType) Update(key, value []byte) bool {return false}

func (b *invalidType) Delete(key []byte) bool {return false}

func (b *invalidType) Lookup(key []byte) ([]byte, error) {return nil, nil}

//func (b *invalidType) WorldState() trie.WorldState {return nil}

func (b *invalidType) Uncles() []core.Byte64 {return nil}

func (b *invalidType) Transactions() []Transaction {return nil}

func (b *invalidType) AddTransaction(tx *Transaction) error {return nil}

func (b *invalidType) Hash() *core.Byte64 {return nil}

func TestSerializeInvalid(t *testing.T) {
	if _, err := serializeBlock(nil); err == nil {
		t.Errorf("Serialization failed to detect nil block")
	}
	if _, err := serializeBlock(&invalidType{}); err == nil {
		t.Errorf("Serialization failed to detect invalid type")
	}
}

func TestSerializeUnHashed(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value"))
	orig := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	if _, err := serializeBlock(orig); err == nil {
		t.Errorf("Serialization failed to detect unhashed block")
	}
}

func TestSerializeInvalidState(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value"))
	orig := newBlock(previous, weight, depth, uint64(0), testMiner, ws)
	orig.computeHash()
	orig.STATE = *core.BytesToByte64([]byte("some random state"))
	if _, err := serializeBlock(orig); err == nil {
		t.Errorf("Serialization failed to detect invalid state")
	}
}

//func TestDeSerializeInvalidState(t *testing.T) {
//	weight, depth := uint64(23), uint64(20)
//	db, _ := db.NewDatabaseInMem()
//	ws := trie.NewMptWorldState(db)
//	ws.Update([]byte("key"), []byte("value"))
//	orig := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
//	orig.computeHash()
//	data, _ := SerializeBlock(orig)
//	// remove state from world state trie
//	oldState := ws.Hash()
//	ws.Update([]byte("key"), []byte("value2"))
//	ws.Cleanup(oldState)
////	if _, err := DeSerializeBlock(data, ws); err == nil {
//	if _, err := DeSerializeBlock(data); err == nil {
//		t.Errorf("Deserialization failed to detect invalid state")
//	}
//}

func TestDeSerializeDecodeError(t *testing.T) {
	if _, err := deSerializeBlock(nil); err == nil {
		t.Errorf("Deserialization failed to detect decode error")
	}
}

func TestTransactionAdd(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := newBlock(previous, weight, depth, now, testMiner, ws)
	if err := b.AddTransaction(testTransaction("transaction 1")); err != nil {
		t.Errorf("failed to add transaction: %s", err)
	}
	if err := b.AddTransaction(testTransaction("transaction 2")); err != nil {
		t.Errorf("failed to add transaction: %s", err)
	}
	if len(b.Transactions()) != 2 {
		t.Errorf("Block transactions list incorrect: Expecting 2, Found '%d'", len(b.Transactions()))
	}
}

func TestTransactionAddDuplicateTransaction(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := newBlock(previous, weight, depth, now, testMiner, ws)
	tx := testTransaction("test transaction")
	// populate DB with transaction
	ws.RegisterTransaction(tx.Id(), core.BytesToByte64([]byte("some random block id")))
	// now attempt to add this transaction to the block
	if err := b.AddTransaction(tx); err == nil {
		t.Errorf("failed to detect pre-existing transaction")
	}
}

func TestUncleAdd(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := newBlock(previous, weight, depth, now, testMiner, ws)
	b.addUncle(testUncle)
	if len(b.Uncles()) != 1 {
		t.Errorf("Block uncle list incorrect: Expecting 1, Found '%d'", len(b.Uncles()))
	}
	if b.Weight().Uint64() != weight+1 {
		t.Errorf("Block weight after uncle add incorrect: Expecting '%d', Found '%d'", weight+1, b.Weight().Uint64())
	}
}