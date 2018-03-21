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
	WorldState() trie.WorldState
	Uncles() []core.Byte64
	Transactions() []Transaction
	AddTransaction(tx *Transaction)
	Hash() *core.Byte64
}

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

func (b *block) WorldState() trie.WorldState {
	return b.worldState
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

func (b *block) AddTransaction(tx *Transaction) {
	b.TXs = append(b.TXs, *tx)
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
	data := make([]byte,0, 64+64+8+64+8+len(b.TXs)*(8+64+1)+8+len(b.UNCLEs)*64+8)
	data = append(data, b.PHASH.Bytes()...)
	data = append(data, b.MINER.Bytes()...)
	data = append(data, b.TS.Bytes()...)
	b.STATE = b.worldState.Hash()
	data = append(data, b.STATE.Bytes()...)
	for _, tx := range b.TXs {
		data = append(data, tx.Bytes()...)
	}
	data = append(data, b.WT.Bytes()...)
	for _, uncle := range b.UNCLEs {
		data = append(data, uncle.Bytes()...)
	}
	data = append(data, b.NONCE.Bytes()...)
	hash := sha512.Sum512(data)
	b.hash = core.BytesToByte64(hash[:])
	return b.hash
}

func NewBlock(previous *core.Byte64, weight uint64, depth uint64, ts uint64, miner *core.Byte64, state trie.WorldState) *block {
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
			STATE: state.Hash(),
			UNCLEs: make([]core.Byte64, 0),
			NONCE: *core.BytesToByte8(nil),
		},
		worldState: state,
		hash: nil,
	}
	return b
}

func SerializeBlock(b Block) ([]byte, error) {
	if b == nil {
		return nil, core.NewCoreError(ERR_INVALID_ARG, "nil block")
	}
	block, ok := b.(*block)
	if !ok {
		return nil, core.NewCoreError(ERR_TYPE_INCORRECT, "incorrect type")
	}
	if block.STATE == *core.BytesToByte64(nil) || block.WorldState() == nil || block.WorldState().Hash() != block.STATE {
		return nil, core.NewCoreError(ERR_STATE_INCORRECT, "block state incorrect")
	}
	if block.hash == nil {
		return nil, core.NewCoreError(ERR_BLOCK_UNHASHED, "block not hashed")
	}
	return common.Serialize(b)
}

func DeSerializeBlock(data []byte, worldState trie.WorldState) (*block, error) {
	var b block
	if err := common.Deserialize(data, &b); err != nil {
		return nil, err
	}
	if err := worldState.Rebase(b.STATE); err != nil {
		return nil, err
	}
	b.worldState = worldState
	b.computeHash()
	return &b, nil
}