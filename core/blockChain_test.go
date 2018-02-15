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
	if now < (chain.td - time.Second * 1) {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", now, chain.TD)
	}
}

func TestBlockChainInMemAddNode(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	block.ComputeHash()
	if err := chain.AddBlockNode(block); err != nil {
		t.Errorf("Failed to add block into chain: '%s'", err)
	}
}

func TestBlockChainInMemAddNodeDepthUpdate(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if chain.Depth() != 1 {
		t.Errorf("Failed to update depth of the chain")
	}
}


func TestBlockChainInMemAddNodeTdUpdate(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if chain.TD() != block.TD() {
		t.Errorf("Failed to update TD of the chain: Expected %d, Actual %d", block.TD(), chain.TD())
	}
}

func TestBlockChainInMemAddNodeUncomputed(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected block with uncomputed hash")
	}
}


func TestBlockChainInMemAddNodeOrphan(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(BytesToByte64([]byte("some random parent hash")), chain.genesis.Hash(), myNode)
	block.ComputeHash()
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected block with non existing parent")
	}
}


func TestBlockChainInMemAddNodeDuplicate(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	// add a block to chain
	block := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	// try adding same block again
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected duplicate block")
	}
}

func TestBlockChainInMemAddNodeForwardLink(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add two child nodes to same ancestor
	child1 := NewSimpleBlock(ancestor.Hash(), chain.genesis.Hash(), myNode)
	child1.ComputeHash()
	chain.AddBlockNode(child1)
	child2 := NewSimpleBlock(ancestor.Hash(), chain.genesis.Hash(), myNode)
	child2.ComputeHash()
	chain.AddBlockNode(child2)
	// verify that ancestor has forward link to children
	parent, _ := chain.BlockNode(ancestor.Hash())
	if len(parent.Children()) != 2 {
		t.Errorf("chain did not update forward links")
	}
	if parent.Children()[0] != child1.Hash() {
		t.Errorf("chain added incorrect forward link 1st child: Expected '%d', Found '%d'", child1.Hash(), parent.Children()[0])
	}
	if parent.Children()[1] != child2.Hash() {
		t.Errorf("chain added incorrect forward link 2nd child: Expected '%d', Found '%d'", child2.Hash(), parent.Children()[1])
	}
}

func makeBlocks(len int, parent *Byte64, genesis *Byte64, miner NodeInfo) []*SimpleBlock {
	nodes := make([]*SimpleBlock, len)
	for i := 0; i < len; i++ {
		nodes[i] = NewSimpleBlock(parent, genesis, miner)
		nodes[i].ComputeHash()
		parent = nodes[i].Hash()
	}
	return nodes
}

func TestBlockChainInMemLongestChain(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add 1st chain with 6 blocks after the ancestor
	chain1 := makeBlocks(6, ancestor.Hash(), chain.genesis.Hash(), myNode)
	for i, child := range(chain1) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("1st chain failed to add block #%d: %s", i, err)
		}
	}
	// now add 2nd chain with 4 blocks after the ancestor
	chain2 := makeBlocks(4, ancestor.Hash(), chain.genesis.Hash(), myNode)
	for i, child := range(chain2) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("2nd chain failed to add block #%d: %s", i, err)
		}
	}
	// validate that longest chain (chain1, length 1+6) wins
	if chain.Depth() != 7 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 7, chain.Depth())
	}
	if chain.TD() != chain1[5].TD() {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", chain1[5].TD(), chain.TD())
	}
}


func TestBlockChainInMemWalkThroughChain(t *testing.T) {
	chain := NewBlockChainInMem()
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add 1st chain with 2 blocks after the ancestor
	chain1 := makeBlocks(2, ancestor.Hash(), chain.genesis.Hash(), myNode)
	for i, child := range(chain1) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("1st chain failed to add block #%d: %s", i, err)
		}
	}
	// now add 2nd chain with 1 blocks after the ancestor
	chain2 := makeBlocks(1, ancestor.Hash(), chain.genesis.Hash(), myNode)
	for i, child := range(chain2) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("2nd chain failed to add block #%d: %s", i, err)
		}
	}
	// ask for list of blocks starting at the ancestor
	blocks := chain.Blocks(ancestor.Hash(), 10)
	if len(blocks) != 4 {
		t.Errorf("chain traversal did not return correct number of blocks: Expected '%d' Found '%d'", 4, len(blocks))
	}
}

