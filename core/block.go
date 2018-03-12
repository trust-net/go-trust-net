package core

import (
	"time"
	"crypto/sha512"
)

// interface for node information 
type NodeInfo interface{
	Id()	 string
}

// interface for block specification
type Block interface {
	ParentHash() *Byte64
	Miner() *Byte64
	Nonce() *Byte8
	Timestamp() *Byte8
	Depth() *Byte8
	Weight() *Byte8
	Uncles() []*Byte64
	OpCode() *Byte8
	Hash() *Byte64
}

// a block spec
type BlockSpec struct {
	ParentHash *Byte64
	Miner *Byte64
	OpCode	*Byte8
	Timestamp *Byte8
	Depth *Byte8
	Weight *Byte8
	Uncles []*Byte64
	Nonce *Byte8
}

// a simple blockchain spec implementation
type SimpleBlock struct {
	parentHash *Byte64
	miner *Byte64
	hash *Byte64
	opCode	*Byte8
	timestamp *Byte8
	depth *Byte8
	weight *Byte8
	uncles []*Byte64
	nonce uint64
}

func (b *SimpleBlock) ParentHash() *Byte64 {
	return b.parentHash
}

func (b *SimpleBlock) Miner() *Byte64 {
	return b.miner
}

func (b *SimpleBlock) Nonce() *Byte8 {
	return Uint64ToByte8(b.nonce)
}

func (b *SimpleBlock) Timestamp() *Byte8 {
	return b.timestamp
}

func (b *SimpleBlock) OpCode() *Byte8 {
	return b.opCode
}

func (b *SimpleBlock) Depth() *Byte8 {
	return b.depth
}

func (b *SimpleBlock) Weight() *Byte8 {
	return b.weight
}

func (b *SimpleBlock) Uncles() []*Byte64 {
	return b.uncles
}

func (b *SimpleBlock) AddTransaction(opCode *Byte8) {
	// we are only doing 1 transaction in a block
	b.opCode = opCode
}

func (b *SimpleBlock) AddUncle(uncle *Byte64, weight uint64) {
	// add uncle to our uncle list
	b.uncles = append(b.uncles, uncle)
	// update weight from uncle's weight
	b.weight = Uint64ToByte8(weight + b.weight.Uint64())
}

func (b *SimpleBlock) Hash() *Byte64 {
	return b.hash
}

type SimpleHeader struct {
	header *Byte64
}

// block hash = SHA512(parent_hash + author_node + timestamp + opcode + nonce)
// we don't do any mining yet!!!
func (b *SimpleBlock) ComputeHash() {
	data := make([]byte,0)
	data = append(data, b.parentHash.Bytes()...)
	data = append(data, b.miner.Bytes()...)
	data = append(data, b.timestamp.Bytes()...)
	data = append(data, b.opCode.Bytes()...)
	for _, uncle := range b.uncles {
		data = append(data, uncle.Bytes()...)
	}
	data = append(data, b.weight.Bytes()...)
	data = append(data, Uint64ToByte8(b.nonce).Bytes()...)
	hash := sha512.Sum512(data)
	b.hash = BytesToByte64(hash[:])
}

type SimpleNodeInfo struct {
	id string
}

func (n *SimpleNodeInfo) Id() string {
	return n.id
}

func NewSimpleNodeInfo(id string) *SimpleNodeInfo {
	return &SimpleNodeInfo{
		id: id,
	}
}

func NewSimpleBlock(previous *Byte64, weight uint64, depth uint64, ts uint64, miner NodeInfo) *SimpleBlock {
	if ts == 0 {
		ts = uint64(time.Now().UnixNano())
	}
	return &SimpleBlock{
		parentHash: previous,
		miner: BytesToByte64([]byte(miner.Id())),
		nonce: 0x0,
		timestamp: Uint64ToByte8(ts),
		depth: Uint64ToByte8(depth),
		weight: Uint64ToByte8(weight),
		uncles: make([]*Byte64, 0, 2),
		opCode: Uint64ToByte8(0),
		hash: nil,
	}
}

// create a new simple block from a block spec
func NewSimpleBlockFromSpec(spec *BlockSpec) *SimpleBlock {
	block := SimpleBlock{
		parentHash: BytesToByte64(spec.ParentHash.Bytes()),
		miner: BytesToByte64(spec.Miner.Bytes()),
		nonce: spec.Nonce.Uint64(),
		timestamp: BytesToByte8(spec.Timestamp.Bytes()),
		depth: BytesToByte8(spec.Depth.Bytes()),
		weight: BytesToByte8(spec.Weight.Bytes()),
		uncles: make([]*Byte64, len(spec.Uncles)),
		opCode: BytesToByte8(spec.OpCode.Bytes()),
		hash: nil,
	}
	for i, uncle := range spec.Uncles {
		block.uncles[i] = BytesToByte64(uncle.Bytes())
	}
	block.ComputeHash()
	return &block
}


// create a new simple block from a block spec
func NewBlockSpecFromBlock(block Block) *BlockSpec {
	spec := BlockSpec{
		ParentHash: BytesToByte64(block.ParentHash().Bytes()),
		Miner: BytesToByte64(block.Miner().Bytes()),
		Nonce: BytesToByte8(block.Nonce().Bytes()),
		Timestamp: BytesToByte8(block.Timestamp().Bytes()),
		Depth: BytesToByte8(block.Depth().Bytes()),
		Weight: BytesToByte8(block.Weight().Bytes()),
		Uncles: make([]*Byte64, len(block.Uncles())),
		OpCode: BytesToByte8(block.OpCode().Bytes()),
	}
	for i, uncle := range block.Uncles() {
		spec.Uncles[i] = BytesToByte64(uncle.Bytes())
	}	
	return &spec
}
