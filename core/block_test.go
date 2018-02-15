package core

import (
    "testing"
    "time"
    "fmt"
    "crypto/sha512"
)

var previous = BytesToByte64([]byte("previous"))
var genesis = BytesToByte64([]byte("genesis"))
func TestNewSimpleBlock(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	now := time.Duration(time.Now().UnixNano())
	block := NewSimpleBlock(previous, genesis, myNode)
	if block.Previous().Bytes() != previous {
		t.Errorf("Block header does not match: Expected '%s', Found '%s'", previous, block.Previous())
	}
	if block.Genesis().Bytes() != genesis {
		t.Errorf("Block header does not match: Expected '%s', Found '%s'", genesis, block.Genesis())
	}
	if block.Miner().Id() != "test node" {
		t.Errorf("Block minder does not match: Expected '%s', Found '%s'", myNode.Id(), block.Miner().Id())
	}
	if block.Nonce() != nil {
		t.Errorf("Block nonce not empty: Found '%s'", block.Nonce())
	}
	if len(block.TXs()) != 0 {
		t.Errorf("Block transaction list not empty: Found '%d'", len(block.TXs()))
	}
	if block.Hash() != nil {
		t.Errorf("Block transaction hash not empty: Found '%s'", block.Hash())
	}
//	if block.Since() != now {
//		t.Errorf("Block genesis time incorrect: Expected '%d' Found '%d'", now, block.Since())
//	}
	if block.TD() > (time.Second + now) {
		t.Errorf("Block total difficulty time incorrect: Found '%d'", block.TD())
	}
}

func TestSimpleBlockHash(t *testing.T) {
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(previous, genesis, myNode)
	block.ComputeHash()
	if len(block.Hash()) != 64 {
		t.Errorf("Block hash not 64 bytes: Found '%d'", block.Hash())
	}
	data := make([]byte,0)
	data = append(data, previous.Bytes()..., )
	data = append(data, []byte(block.miner.Id())...)
	data = append(data, genesis.Bytes()...)
	data = append(data, []byte(fmt.Sprintf("%d",block.td))...)
	hash := sha512.Sum512(data)
	if *block.Hash() != *BytesToByte64(hash[:]) {
		t.Errorf("Block hash incorrect: Expected '%d' Found '%d'", BytesToByte64(hash[:]), block.Hash())
	}
}