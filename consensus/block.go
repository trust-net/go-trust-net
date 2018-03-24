package consensus

import (
	"time"
	"crypto/sha512"
	"encoding/gob"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/core/trie"
	"github.com/trust-net/go-trust-net/common"
)

type Block interface {
	ParentHash() *core.Byte64
	Miner() *core.Byte64
	Nonce() *core.Byte8
	Timestamp() *core.Byte8
	Depth() *core.Byte8
	Weight() *core.Byte8
	Update(key, value []byte) bool
	Delete(key []byte) bool
	Lookup(key []byte) ([]byte, error)
	Uncles() []core.Byte64
	Transactions() []Transaction
	AddTransaction(tx *Transaction) error
	Hash() *core.Byte64
}

// these are the fields that actually go over the wire
type BlockSpec struct {
	PHASH core.Byte64
	MINER core.Byte64
	STATE core.Byte64
	TXs []Transaction
	TS core.Byte8
	DEPTH core.Byte8
	WT core.Byte8
	UNCLEs []core.Byte64
	NONCE core.Byte8
}

func init() {
	gob.Register(&BlockSpec{})
	gob.Register(&block{})
}

type block struct {
	BlockSpec
	worldState trie.WorldState
	hash *core.Byte64
	isNetworkBlock bool
}

func (b *block) ParentHash() *core.Byte64 {
	return &b.PHASH
}

func (b *block) Miner() *core.Byte64 {
	return &b.MINER
}

func (b *block) Nonce() *core.Byte8 {
	return &b.NONCE
}

func (b *block) Timestamp() *core.Byte8 {
	return &b.TS
}

func (b *block) Depth() *core.Byte8 {
	return &b.DEPTH
}

func (b *block) Weight() *core.Byte8 {
	return &b.WT
}

func (b *block) Update(key, value []byte) bool {
	hash := b.worldState.Hash()
	return b.worldState.Update(key, value) != hash
}

func (b *block) Delete(key []byte) bool {
	hash := b.worldState.Hash()
	return b.worldState.Delete(key) != hash
}

func (b *block) Lookup(key []byte) ([]byte, error) {
	return b.worldState.Lookup(key)
}

func (b *block) Uncles() []core.Byte64 {
	return b.UNCLEs
}

func (b *block) addUncle(uncle *core.Byte64) {
	b.UNCLEs = append(b.UNCLEs, *uncle)
	b.WT = *core.Uint64ToByte8(b.WT.Uint64()+1)
}

func (b *block) Transactions() []Transaction {
	return b.TXs
}

func (b *block) AddTransaction(tx *Transaction) error {
	// first check if transaction does not already exists
	if _, err := b.worldState.HasTransaction(tx.Id()); err == nil {
		return core.NewCoreError(ERR_DUPLICATE_TX, "duplicate transaction")
	}
	// accept transaction to the list 
	b.TXs = append(b.TXs, *tx)
	// not updating world state with transactions yet because don't have hash computed yet,
	// this will be done after hash computation, in the computeHash method
	return nil
}

func (b *block) Hash() *core.Byte64 {
	return b.hash
}

// block hash = SHA512(parent_hash + author_node + timestamp + state + depth + transactions... + weight + uncles... + nonce)
func (b *block) computeHash() *core.Byte64 {
	// parent hash +
	// miner ID +
	// block timestampt +
	// world state fingerprint +
	// block's depth from genesis +
	// transactions... +
	// block's weight +
	// block's uncle's hash +
	// nonce
	data := make([]byte,0, 64+64+8+64+8+len(b.TXs)*(8+64+1)+8+len(b.UNCLEs)*64)
	data = append(data, b.PHASH.Bytes()...)
	data = append(data, b.MINER.Bytes()...)
	data = append(data, b.TS.Bytes()...)
	if b.worldState != nil {
		b.STATE = b.worldState.Hash()
	}
	data = append(data, b.STATE.Bytes()...)
	for _, tx := range b.TXs {
		data = append(data, tx.Bytes()...)
	}
	data = append(data, b.WT.Bytes()...)
	for _, uncle := range b.UNCLEs {
		data = append(data, uncle.Bytes()...)
	}
	dataWithNonce := make([]byte, 0, len(data)+8)
	nonce := b.NONCE.Uint64()
	var hash [sha512.Size]byte
	isPoWDone := false

	for !isPoWDone {
		// TODO: run the PoW
		b.NONCE = *core.Uint64ToByte8(nonce)
		nonce++
		dataWithNonce = append(data, b.NONCE.Bytes()...)
		hash = sha512.Sum512(dataWithNonce)
		// check PoW validation
		// TODO
		isPoWDone = true
		
		// if a network block, then 1st hash MUST be correct
		if !isPoWDone && b.isNetworkBlock {
			// return an error
			return nil
		}
	}
	b.hash = core.BytesToByte64(hash[:])
	// update the world state with this block's transactions
	if b.worldState != nil {
		for _, tx := range b.TXs {
			if err := b.worldState.RegisterTransaction(tx.Id(), b.hash); err != nil {
				b.hash = nil
				break
			}
		}
	}
	return b.hash
}

// create a copy of block
func (b *block) clone(state trie.WorldState) *block {
	return &block{
		BlockSpec: BlockSpec {
			PHASH: b.PHASH,
			MINER: b.MINER,
			TXs: nil,
			TS: b.TS,
			DEPTH: b.DEPTH,
			WT: b.WT,
			UNCLEs: nil,
			NONCE: b.NONCE,
		},
		hash: b.hash,
		worldState: state,
		isNetworkBlock: false,
	}
}

// private method, can only be invoked by DAG implementation, so can be initiaized correctly
func newBlock(previous *core.Byte64, weight uint64, depth uint64, ts uint64, miner *core.Byte64, state trie.WorldState) *block {
	if ts == 0 {
		ts = uint64(time.Now().UnixNano())
	}
	b := &block{
		BlockSpec: BlockSpec {
			PHASH: *previous,
			MINER: *miner,
			TXs: make([]Transaction,0,1),
			TS: *core.Uint64ToByte8(ts),
			DEPTH: *core.Uint64ToByte8(depth),
			WT: *core.Uint64ToByte8(weight),
//			STATE: state.Hash(),
			UNCLEs: make([]core.Byte64, 0),
			NONCE: *core.BytesToByte8(nil),
		},
		worldState: state,
		hash: nil,
	}
	if state != nil {
		b.STATE = state.Hash()
	}
	return b
}

func serializeBlock(b Block) ([]byte, error) {
	if b == nil {
		return nil, core.NewCoreError(ERR_INVALID_ARG, "nil block")
	}
	block, ok := b.(*block)
	if !ok {
		return nil, core.NewCoreError(ERR_TYPE_INCORRECT, "incorrect type")
	}
	if block.STATE == *core.BytesToByte64(nil) || block.worldState == nil || block.worldState.Hash() != block.STATE {
		return nil, core.NewCoreError(ERR_STATE_INCORRECT, "block state incorrect")
	}
	if block.hash == nil {
		return nil, core.NewCoreError(ERR_BLOCK_UNHASHED, "block not hashed")
	}
	return common.Serialize(b)
}

// private method, can only be invoked by DAG implementation, so that world state can be added after deserialization
// here we only want to deserialize wire protocol data into block instance, then after this DAG implementation
// will use a world state rebased to parent's state trie, and then pass it on to application to run
// the transactions and value changes as appropriate
func deSerializeBlock(data []byte) (*block, error) {
	var b block
	if err := common.Deserialize(data, &b); err != nil {
		return nil, err
	}
	b.computeHash()

	// Q: when, where, who to update world state with this block's value changes?
	// A: application will validate transactions, at which time world state will be updated with values
	//    and then submit the network block for acceptance, at which time canonical chain will start pointing
	//    to world state view of this block (if accepted)
	return &b, nil
}