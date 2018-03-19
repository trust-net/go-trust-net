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
	weight, depth := uint64(23), uint64(20)
	block := NewSimpleBlock(previous, weight, depth, now, myNode)
	if block.ParentHash() != previous {
		t.Errorf("Block header does not match: Expected '%s', Found '%s'", previous, block.ParentHash())
	}
	if *block.Miner() != *BytesToByte64([]byte(myNode.Id())) {
		t.Errorf("Block miner does not match: Expected '%s', Found '%s'", BytesToByte64([]byte(myNode.Id())), block.Miner())
	}
	if block.Nonce().Uint64() != uint64(0x0) {
		t.Errorf("Block nonce not initialized: Found '%s'", block.Nonce())
	}
	if len(block.Transactions()) != 0 {
		t.Errorf("Block's transactions not empty: Found '%x'", block.Transactions())
	}
	if block.Weight().Uint64() != weight {
		t.Errorf("Block's weight does not match: Expected '%d', Found '%d'", weight, block.Weight().Uint64())
	}
	if block.Depth().Uint64() != depth {
		t.Errorf("Block's depth does not match: Expected '%d', Found '%d'", depth, block.Depth().Uint64())
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
	weight, depth := uint64(23), uint64(20)
	block := NewSimpleBlock(previous, weight, depth, now, myNode)
	// add first transaction
	block.AddTransaction(Uint64ToByte8(0x123))
	if block.Transactions()[0].Uint64() != 0x123 {
		t.Errorf("Block's op code not set: Expected '%d', Found '%d'", 0x123, block.Transactions()[0])
	}
	// add second transaction
	block.AddTransaction(Uint64ToByte8(0x456))
	if block.Transactions()[1].Uint64() != 0x456 {
		t.Errorf("Block's op code not updated: Expected '%d', Found '%d'", 0x456, block.Transactions()[1])
	}	
}

func TestSimpleBlockAddUncle(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	block := NewSimpleBlock(previous, weight, depth, now, myNode)
	// add uncle
	uncleHash, uncleWeight := BytesToByte64(Uint64ToByte8(0x1234).Bytes()), uint64(0x1234)
	block.AddUncle(uncleHash, uncleWeight)
	// check if block updated its weight
	if block.Weight().Uint64() != (weight + uncleWeight) {
		t.Errorf("Block's weight not updated: Expected '%d', Found '%d'", weight + uncleWeight, block.Weight().Uint64())
	}
	// check if block added uncle
	uncles := block.Uncles()
	if len(uncles) != 1 || *uncles[0] != *uncleHash {
		t.Errorf("Block did not updated uncles: Expected '%d', Found '%d'", 1, len(uncles))
	}
}

func TestSimpleBlockHash(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	weight, depth := uint64(23), uint64(20)
	block := NewSimpleBlock(previous, weight, depth, 0, myNode)
	uncle1hash, uncle1weight := BytesToByte64([]byte("uncle 1")), uint64(2)
	uncle2hash, uncle2weight := BytesToByte64([]byte("uncle 2")), uint64(2)
	block.AddUncle(uncle1hash, uncle1weight)
	block.AddUncle(uncle2hash, uncle2weight)
	block.AddTransaction(Uint64ToByte8(0x01))
	block.ComputeHash()
	if len(block.Hash()) != 64 {
		t.Errorf("Block hash not 64 bytes: Found '%d'", block.Hash())
	}
	// block hash = SHA512(parent_hash + author_node + timestamp + opcode + nonce)
	data := make([]byte,0)
	data = append(data, previous.Bytes()..., )
	data = append(data, block.miner.Bytes()...)
	data = append(data, block.timestamp.Bytes()...)
	data = append(data, block.state.Bytes()...)
	data = append(data, block.Transactions()[0].Bytes()...)
	data = append(data, Uint64ToByte8(weight+uncle1weight+uncle2weight).Bytes()...)
	data = append(data, uncle1hash.Bytes()...)
	data = append(data, uncle2hash.Bytes()...)
	data = append(data, block.Nonce().Bytes()...)
	hash := sha512.Sum512(data)
	if *block.Hash() != *BytesToByte64(hash[:]) {
		t.Errorf("Block hash incorrect: Expected '%d' Found '%d'", BytesToByte64(hash[:]), block.Hash())
	}
}

func TestNewSimpleBlockFromSpec(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := uint64(time.Now().UnixNano())
	weight, depth := uint64(23), uint64(20)
	spec := BlockSpec {
		ParentHash: previous,
		Miner: BytesToByte64([]byte(myNode.Id())),
		Nonce: Uint64ToByte8(0x123456),
		Timestamp: Uint64ToByte8(now),
		Weight: Uint64ToByte8(weight),
		Depth: Uint64ToByte8(depth),
		Uncles: []*Byte64{BytesToByte64([]byte("uncle1")), BytesToByte64([]byte("uncle2"))},
		Transactions: []*Byte8{Uint64ToByte8(0x08)},
		State: BytesToByte64(nil),
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
	if block.Transactions()[0].Uint64() != uint64(0x08) {
		t.Errorf("Block's op code not correct: Expected '%d', Found '%d'", 0x08, block.Transactions()[0].Uint64())
	}
	if block.Weight().Uint64() != weight {
		t.Errorf("Block's weight does not match: Expected '%d', Found '%d'", weight, block.Weight().Uint64())
	}
	if block.Depth().Uint64() != depth {
		t.Errorf("Block's depth does not match: Expected '%d', Found '%d'", depth, block.Depth().Uint64())
	}
	if len(block.Uncles()) != 2 {
		t.Errorf("Block's uncles count not correct: Expected '%d', Found '%d'", 2, len(block.Uncles()))
	}
	if *block.Uncles()[0] != *BytesToByte64([]byte("uncle1")) {
		t.Errorf("Block's uncle hash not correct: Expected '%x', Found '%x'", *BytesToByte64([]byte("uncle1")), *block.Uncles()[0])
	}
	if *block.Uncles()[1] != *BytesToByte64([]byte("uncle2")) {
		t.Errorf("Block's uncle hash not correct: Expected '%x', Found '%x'", *BytesToByte64([]byte("uncle2")), *block.Uncles()[1])
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
	weight, depth := uint64(23), uint64(20)
	block := NewSimpleBlock(previous, weight, depth, 0, myNode)
	block.AddUncle(BytesToByte64([]byte("uncle1")), 1)
	block.AddUncle(BytesToByte64([]byte("uncle2")), 2)
	block.nonce = 0x123456
	block.AddTransaction(Uint64ToByte8(0x08))
	spec := NewBlockSpecFromBlock(block)
	if *spec.Miner != *BytesToByte64([]byte(myNode.Id())) {
		t.Errorf("Spec's miner does not match: Expected '%s', Found '%s'", BytesToByte64([]byte(myNode.Id())), spec.Miner)
	}
	if spec.Nonce.Uint64() != uint64(0x123456) {
		t.Errorf("Spec's nonce not initialized: Expected '%d', Found '%d'", 0x123456, spec.Nonce.Uint64())
	}
	if spec.Transactions[0].Uint64() != uint64(0x08) {
		t.Errorf("Spec's op code not correct: Expected '%d', Found '%d'", 0x08, spec.Transactions[0].Uint64())
	}
	if spec.Timestamp.Uint64() != block.timestamp.Uint64() {
		t.Errorf("Spec's time stamp incorrect: Expected '%d', Found '%d'", block.timestamp.Uint64(), spec.Timestamp.Uint64())
	}
	if spec.Weight.Uint64() != weight+1+2 {
		t.Errorf("Spec's weight does not match: Expected '%d', Found '%d'", weight+1+2, spec.Weight.Uint64())
	}
	if spec.Depth.Uint64() != depth {
		t.Errorf("Spec's depth does not match: Expected '%d', Found '%d'", depth, spec.Depth.Uint64())
	}
	if len(spec.Uncles) != 2 {
		t.Errorf("Spec's uncles count not correct: Expected '%d', Found '%d'", 2, len(spec.Uncles))
	}
	if *spec.Uncles[0] != *BytesToByte64([]byte("uncle1")) {
		t.Errorf("Spec's uncle hash not correct: Expected '%x', Found '%x'", *BytesToByte64([]byte("uncle1")), *spec.Uncles[0])
	}
	if *spec.Uncles[1] != *BytesToByte64([]byte("uncle2")) {
		t.Errorf("Spec's uncle hash not correct: Expected '%x', Found '%x'", *BytesToByte64([]byte("uncle2")), *spec.Uncles[1])
	}
}