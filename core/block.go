package core

import (
	"time"
	"crypto/sha512"
)

// interface to implement transactions
type Transaction interface {
	// execute using input data, and produce and output or error
	Execute(input interface{}) (interface{}, error)
}

// interface to implement block headers
type Header interface {
	Bytes() *Byte64
}

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
	OpCode() *Byte8
	Hash() *Byte64
}

// a simple blockchain spec implementation
type SimpleBlock struct {
	parentHash *Byte64
	miner *Byte64
	hash *Byte64
	opCode	*Byte8
	timestamp *Byte8
	nonce *Byte8
}

func (b *SimpleBlock) ParentHash() *Byte64 {
	return b.parentHash
}

func (b *SimpleBlock) Miner() *Byte64 {
	return b.miner
}

func (b *SimpleBlock) Nonce() *Byte8 {
	return b.nonce
}

func (b *SimpleBlock) Timestamp() *Byte8 {
	return b.timestamp
}

func (b *SimpleBlock) OpCode() *Byte8 {
	return b.opCode
}

func (b *SimpleBlock) Hash() *Byte64 {
	return b.hash
}

type SimpleHeader struct {
	header *Byte64
}

func (h *SimpleHeader) Bytes() *Byte64 {
	return h.header
}

func NewSimpleHeader(header *Byte64) *SimpleHeader {
	return &SimpleHeader {
		header: header,
	}
}

// block hash = SHA512(parent_hash + author_node + timestamp + opcode + nonce)
func (b *SimpleBlock) ComputeHash() {
	data := make([]byte,0)
	data = append(data, b.parentHash.Bytes()...)
	data = append(data, b.miner.Bytes()...)
	data = append(data, b.timestamp.Bytes()...)
	data = append(data, b.opCode.Bytes()...)
	data = append(data, b.nonce.Bytes()...)
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

func NewSimpleBlock(previous *Byte64, miner NodeInfo) *SimpleBlock {
	return &SimpleBlock{
		parentHash: previous,
		miner: BytesToByte64([]byte(miner.Id())),
		nonce: Uint64ToByte8(0x0),
		timestamp: Uint64ToByte8(uint64(time.Now().UnixNano())),
		opCode: Uint64ToByte8(0),
		hash: nil,
	}
}
