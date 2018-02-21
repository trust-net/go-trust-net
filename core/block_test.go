package core

import (
    "testing"
    "time"
    "crypto/sha512"
)

var previous = BytesToByte64([]byte("previous"))
var genesis = BytesToByte64([]byte("genesis"))
func TestNewSimpleBlock(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := uint64(time.Now().UnixNano())
	block := NewSimpleBlock(previous, now, myNode)
	if block.ParentHash() != previous {
		t.Errorf("Block header does not match: Expected '%s', Found '%s'", previous, block.ParentHash())
	}
	if *block.Miner() != *BytesToByte64([]byte(myNode.Id())) {
		t.Errorf("Block miner does not match: Expected '%s', Found '%s'", BytesToByte64([]byte(myNode.Id())), block.Miner())
	}
	if block.Nonce().Uint64() != uint64(0x0) {
		t.Errorf("Block nonce not initialized: Found '%s'", block.Nonce())
	}
	if block.OpCode().Uint64() != 0 {
		t.Errorf("Block's op code not empty: Found '%d'", block.OpCode())
	}
	if block.Hash() != nil {
		t.Errorf("Block transaction hash not empty: Found '%s'", block.Hash())
	}
	if block.Timestamp().Uint64() != now {
		t.Errorf("Block time stamp incorrect: Expected '%d', Found '%d'", now, block.Timestamp().Uint64())
	}
}

func TestSimpleBlockAddTransaction(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := uint64(time.Now().UnixNano())
	block := NewSimpleBlock(previous, now, myNode)
	// add first transaction
	block.AddTransaction(Uint64ToByte8(0x123))
	if block.OpCode().Uint64() != 0x123 {
		t.Errorf("Block's op code not set: Expected '%d', Found '%d'", 0x123, block.OpCode())
	}
	// add second transaction
	block.AddTransaction(Uint64ToByte8(0x456))
	if block.OpCode().Uint64() != 0x456 {
		t.Errorf("Block's op code not updated: Expected '%d', Found '%d'", 0x456, block.OpCode())
	}	
}
func TestSimpleBlockHash(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(previous, 0, myNode)
	block.ComputeHash()
	if len(block.Hash()) != 64 {
		t.Errorf("Block hash not 64 bytes: Found '%d'", block.Hash())
	}
	// block hash = SHA512(parent_hash + author_node + timestamp + opcode + nonce)
	data := make([]byte,0)
	data = append(data, previous.Bytes()..., )
	data = append(data, block.miner.Bytes()...)
	data = append(data, block.timestamp.Bytes()...)
	data = append(data, block.opCode.Bytes()...)
	data = append(data, block.Nonce().Bytes()...)
	hash := sha512.Sum512(data)
	if *block.Hash() != *BytesToByte64(hash[:]) {
		t.Errorf("Block hash incorrect: Expected '%d' Found '%d'", BytesToByte64(hash[:]), block.Hash())
	}
}

func TestNewSimpleBlockFromSpec(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := uint64(time.Now().UnixNano())
	spec := BlockSpec {
		ParentHash: previous,
		Miner: BytesToByte64([]byte(myNode.Id())),
		Nonce: Uint64ToByte8(0x123456),
		Timestamp: Uint64ToByte8(now),
		OpCode: Uint64ToByte8(0x08),
	}
	block := NewSimpleBlockFromSpec(&spec)
	if *block.ParentHash() != *previous {
		t.Errorf("Block previous hash does not match: Expected '%s', Found '%s'", previous, block.ParentHash())
	}
	if *block.Miner() != *BytesToByte64([]byte(myNode.Id())) {
		t.Errorf("Block miner does not match: Expected '%s', Found '%s'", BytesToByte64([]byte(myNode.Id())), block.Miner())
	}
	if block.nonce != uint64(0x123456) {
		t.Errorf("Block nonce not initialized: Expected '%d', Found '%d'", 0x123456, block.nonce)
	}
	if block.OpCode().Uint64() != uint64(0x08) {
		t.Errorf("Block's op code not correct: Expected '%d', Found '%d'", 0x08, block.OpCode().Uint64())
	}
	if block.Hash() == nil {
		t.Errorf("Block did not compute hash from spec")
	}
	if block.Timestamp().Uint64() != now {
		t.Errorf("Block time stamp incorrect: Expected '%d', Found '%d'", now, block.Timestamp().Uint64())
	}
}

func TestNewBlockSpecFromBlock(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(previous, 0, myNode)
	block.nonce = 0x123456
	block.opCode = Uint64ToByte8(0x08)
	spec := NewBlockSpecFromBlock(block)
	if *spec.Miner != *BytesToByte64([]byte(myNode.Id())) {
		t.Errorf("Block miner does not match: Expected '%s', Found '%s'", BytesToByte64([]byte(myNode.Id())), spec.Miner)
	}
	if spec.Nonce.Uint64() != uint64(0x123456) {
		t.Errorf("Block nonce not initialized: Expected '%d', Found '%d'", 0x123456, spec.Nonce.Uint64())
	}
	if spec.OpCode.Uint64() != uint64(0x08) {
		t.Errorf("Block's op code not correct: Expected '%d', Found '%d'", 0x08, spec.OpCode.Uint64())
	}
	if spec.Timestamp.Uint64() != block.timestamp.Uint64() {
		t.Errorf("Block time stamp incorrect: Expected '%d', Found '%d'", block.timestamp.Uint64(), spec.Timestamp.Uint64())
	}
}