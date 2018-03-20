package consensus

import (
    "testing"
    "time"
//    "crypto/sha512"
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
	b := NewBlock(previous, weight, depth, now, testMiner, ws)
	if b.Timestamp().Uint64() != now {
		t.Errorf("Block time stamp incorrect: Expected '%d', Found '%d'", now, b.Timestamp().Uint64())
	}
}

func TestNewBlockWithoutTime(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	before := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
	after := uint64(time.Now().UnixNano())
	if b.Timestamp().Uint64() < before {
		t.Errorf("Block time stamp incorrect: Expected > '%d', Found '%d'", before, b.Timestamp().Uint64())
	}
	if b.Timestamp().Uint64() > after {
		t.Errorf("Block time stamp incorrect: Expected < '%d', Found '%d'", after, b.Timestamp().Uint64())
	}
}

func TestNewBlockDefaultInitializations(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	b := NewBlock(previous, 0, 0, uint64(0), testMiner, ws)
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
	if b.WorldState().Hash() != ws.Hash() {
		t.Errorf("Block worldstate not correct:\nExpected '%x\nFound '%x'", ws.Hash(), b.WorldState().Hash())
	}
}

func TestSerializeDeserializeOnTheWire(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	orig := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
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

func TestSerializeDeserialize(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value"))
	orig := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
	orig.AddTransaction(testTransaction("transaction 1"))
	orig.AddTransaction(testTransaction("transaction 2"))
	orig.addUncle(testUncle)
	orig.computeHash()
	var copy Block
	if data, err := SerializeBlock(orig); err != nil {
		t.Errorf("Failed to serialize block: %s", err)
		return
	} else {
		if copy, err = DeSerializeBlock(data, ws); err != nil {
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
	if copy.WorldState().Hash() != ws.Hash() {
		t.Errorf("Block worldState not initialized: Expected '%x', Found '%x'", ws.Hash(), copy.WorldState().Hash())
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

type invalidType struct {}

func (b *invalidType) ParentHash() *core.Byte64 {return nil}

func (b *invalidType) Miner() *core.Byte64 {return nil}

func (b *invalidType) Nonce() *core.Byte8 {return nil}

func (b *invalidType) Timestamp() *core.Byte8 {return nil}

func (b *invalidType) Depth() *core.Byte8 {return nil}

func (b *invalidType) Weight() *core.Byte8 {return nil}

func (b *invalidType) WorldState() trie.WorldState {return nil}

func (b *invalidType) Uncles() []core.Byte64 {return nil}

func (b *invalidType) Transactions() []Transaction {return nil}

func (b *invalidType) AddTransaction(tx *Transaction) {return}

func (b *invalidType) Hash() *core.Byte64 {return nil}

func TestSerializeInvalid(t *testing.T) {
	if _, err := SerializeBlock(nil); err == nil {
		t.Errorf("Serialization failed to detect nil block")
	}
	if _, err := SerializeBlock(&invalidType{}); err == nil {
		t.Errorf("Serialization failed to detect invalid type")
	}
}

func TestSerializeUnHashed(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value"))
	orig := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
	if _, err := SerializeBlock(orig); err == nil {
		t.Errorf("Serialization failed to detect unhashed block")
	}
}

func TestSerializeInvalidState(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value"))
	orig := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
	orig.computeHash()
	orig.STATE = *core.BytesToByte64([]byte("some random state"))
	if _, err := SerializeBlock(orig); err == nil {
		t.Errorf("Serialization failed to detect invalid state")
	}
}

func TestDeSerializeInvalidState(t *testing.T) {
	weight, depth := uint64(23), uint64(20)
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	ws.Update([]byte("key"), []byte("value"))
	orig := NewBlock(previous, weight, depth, uint64(0), testMiner, ws)
	orig.computeHash()
	data, _ := SerializeBlock(orig)
	// remove state from world state trie
	oldState := ws.Hash()
	ws.Update([]byte("key"), []byte("value2"))
	ws.Cleanup(oldState)
	if _, err := DeSerializeBlock(data, ws); err == nil {
		t.Errorf("Deserialization failed to detect invalid state")
	}
}

func TestDeSerializeDecodeError(t *testing.T) {
	if _, err := DeSerializeBlock(nil, nil); err == nil {
		t.Errorf("Deserialization failed to detect decode error")
	}
}

func TestTransactionAdd(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := NewBlock(previous, weight, depth, now, testMiner, ws)
	b.AddTransaction(testTransaction("transaction 1"))
	b.AddTransaction(testTransaction("transaction 2"))
	if len(b.Transactions()) != 2 {
		t.Errorf("Block transactions list incorrect: Expecting 2, Found '%d'", len(b.Transactions()))
	}
}


func TestUncleAdd(t *testing.T) {
	db, _ := db.NewDatabaseInMem()
	ws := trie.NewMptWorldState(db)
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	b := NewBlock(previous, weight, depth, now, testMiner, ws)
	b.addUncle(testUncle)
	if len(b.Uncles()) != 1 {
		t.Errorf("Block uncle list incorrect: Expecting 1, Found '%d'", len(b.Uncles()))
	}
	if b.Weight().Uint64() != weight+1 {
		t.Errorf("Block weight after uncle add incorrect: Expecting '%d', Found '%d'", weight+1, b.Weight().Uint64())
	}
}