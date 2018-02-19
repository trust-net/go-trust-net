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
	now := time.Duration(time.Now().UnixNano())
	block := NewSimpleBlock(previous, myNode)
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
//	if block.Since() != now {
//		t.Errorf("Block genesis time incorrect: Expected '%d' Found '%d'", now, block.Since())
//	}
	if block.Timestamp().Uint64() > uint64(time.Second + now) {
		t.Errorf("Block total difficulty time incorrect: Found '%d'", block.Timestamp())
	}
}

func TestSimpleBlockHash(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(previous, myNode)
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
	data = append(data, block.nonce.Bytes()...)
	hash := sha512.Sum512(data)
	if *block.Hash() != *BytesToByte64(hash[:]) {
		t.Errorf("Block hash incorrect: Expected '%d' Found '%d'", BytesToByte64(hash[:]), block.Hash())
	}
}