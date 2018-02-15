package core

import (
    "testing"
    "time"
)

func TestNewBlockChainInMem(t *testing.T) {
	now := time.Duration(time.Now().UnixNano())
	chain := NewBlockChainInMem()
	if chain.depth != 0 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 0, chain.depth)
	}
	if chain.genesis.Depth() != 0 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 0, chain.depth)
	}
	if now < (chain.TD - time.Second * 1) {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", now, chain.TD)
	}
}

