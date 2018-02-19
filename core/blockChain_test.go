package core

import (
    "testing"
    "time"
    "sync"
    "math/rand"
	"github.com/trust-net/go-trust-net/log"
)

func testGenesisBlock() Block{
	b := NewSimpleBlock(BytesToByte64(nil), NewSimpleNodeInfo(""))
	b.ComputeHash()
	return b
}


func TestNewBlockChainInMem(t *testing.T) {
	log.SetLogLevel(log.NONE)
	now := uint64(time.Now().UnixNano())
	chain := NewBlockChainInMem(testGenesisBlock())
	if chain.depth != 0 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 0, chain.depth)
	}
	if chain.genesis.Depth() != 0 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 0, chain.depth)
	}
	if now < (chain.td.Uint64() - uint64(time.Second * 1)) {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", now, chain.TD())
	}
	bn,_ := chain.BlockNode(chain.genesis.Hash())
	if !bn.IsMainList() {
		t.Errorf("did not set main list flag on genesis node")
	}
	if chain.Tip().Hash() != bn.Hash() {
		t.Errorf("did not set block chain tip to genesis node")
	}
	if chain.Tip().Depth() != 0 {
		t.Errorf("did not set depth of genesis node as 0")
	}
}

func TestBlockChainInMemAddNode(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), myNode)
	block.ComputeHash()
	if err := chain.AddBlockNode(block); err != nil {
		t.Errorf("Failed to add block into chain: '%s'", err)
	}
	bn,_ := chain.BlockNode(block.Hash())
	if !bn.IsMainList() {
		t.Errorf("did not update main list flag on new node")
	}
	if bn.Parent() != chain.genesis.Hash() {
		t.Errorf("did not set parent link on new node")
	}
}

func TestBlockChainInMemAddNodeDepthUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if chain.Depth() != 1 {
		t.Errorf("Failed to update depth of the chain")
	}
}


func TestBlockChainInMemAddNodeTdUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if *chain.TD() != *block.Timestamp() {
		t.Errorf("Failed to update TD of the chain: Expected %d, Actual %d", block.Timestamp(), chain.TD())
	}
}

func TestBlockChainInMemAddNodeUncomputed(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(chain.genesis.Hash(), myNode)
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected block with uncomputed hash")
	}
}

func TestBlockChainInMemAddNodeNil(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
//	myNode := NewSimpleNodeInfo("test node")
//	block := NewSimpleBlock(chain.genesis.Hash(), chain.genesis.Hash(), myNode)
	if err := chain.AddBlockNode(nil); err == nil {
		t.Errorf("Failed to detected nil block")
	}
}

func TestBlockChainInMemAddNodeOrphan(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	block := NewSimpleBlock(BytesToByte64([]byte("some random parent hash")), myNode)
	block.ComputeHash()
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected block with non existing parent")
	}
}


func TestBlockChainInMemAddNodeDuplicate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	// add a block to chain
	block := NewSimpleBlock(chain.genesis.Hash(), myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	// try adding same block again
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected duplicate block")
	}
}

func TestBlockChainInMemAddNodeForwardLink(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add two child nodes to same ancestor
	child1 := NewSimpleBlock(ancestor.Hash(), myNode)
	child1.ComputeHash()
	chain.AddBlockNode(child1)
	child2 := NewSimpleBlock(ancestor.Hash(), myNode)
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

func makeBlocks(len int, parent *Byte64, miner NodeInfo) []*SimpleBlock {
	nodes := make([]*SimpleBlock, len)
	for i := 0; i < len; i++ {
		nodes[i] = NewSimpleBlock(parent,  miner)
		nodes[i].ComputeHash()
		parent = nodes[i].Hash()
	}
	return nodes
}

func TestBlockChainInMemLongestChain(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add 1st chain with 6 blocks after the ancestor
	chain1 := makeBlocks(6, ancestor.Hash(), myNode)
	for i, child := range(chain1) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("1st chain failed to add block #%d: %s", i, err)
		}
	}
	// now add 2nd chain with 4 blocks after the ancestor
	chain2 := makeBlocks(4, ancestor.Hash(), myNode)
	for i, child := range(chain2) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("2nd chain failed to add block #%d: %s", i, err)
		}
	}
	// validate that longest chain (chain1, length 1+6) wins
	if chain.Depth() != 7 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 7, chain.Depth())
	}
	if *chain.TD() != *chain1[5].Timestamp() {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", chain1[5].Timestamp(), chain.TD())
	}
}

func TestBlockChainInMemRebalance(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// add 1st child to ancestor
	child1 := NewSimpleBlock(ancestor.Hash(), myNode)
	child1.ComputeHash()
	chain.AddBlockNode(child1)
	// add 2nd child to ancestor
	child2 := NewSimpleBlock(ancestor.Hash(), myNode)
	child2.ComputeHash()
	chain.AddBlockNode(child2)
	// verify that blockchain points to 1st child as tip
	if chain.Tip().Hash() != child1.Hash() {
		t.Errorf("chain tip incorrect: Expected '%d' Found '%d'", child1, chain.Tip())
	}
	bn,_ := chain.BlockNode(child1.Hash())
	if !bn.IsMainList() {
		t.Errorf("chain tip incorrect: Expected child1 in mainlist '%d' Found '%d'", true, bn.IsMainList())
	}
	bn,_ = chain.BlockNode(child2.Hash())
	if bn.IsMainList() {
		t.Errorf("chain tip incorrect: Expected child2 in mainlist '%d' Found '%d'", false, bn.IsMainList())
	}
	
	// now add new block to child2 (current non main list)
	child3 := NewSimpleBlock(child2.Hash(), myNode)
	child3.ComputeHash()
	chain.AddBlockNode(child3)
	// verify that blockchain points to 3rd child as tip
	if chain.Tip().Hash() != child3.Hash() {
		t.Errorf("chain tip incorrect: Expected '%d' Found '%d'", child3, chain.Tip())
	}
	// confirm that main list flags have been rebalanced
	bn,_ = chain.BlockNode(child1.Hash())
	if bn.IsMainList() {
		t.Errorf("chain tip incorrect: Expected child1 in mainlist '%d' Found '%d'", false, bn.IsMainList())
	}
	bn,_ = chain.BlockNode(child2.Hash())
	if !bn.IsMainList() {
		t.Errorf("chain tip incorrect: Expected child2 in mainlist '%d' Found '%d'", true, bn.IsMainList())
	}
	bn,_ = chain.BlockNode(child3.Hash())
	if !bn.IsMainList() {
		t.Errorf("chain tip incorrect: Expected child3 in mainlist '%d' Found '%d'", true, bn.IsMainList())
	}
}

func TestBlockChainInMemWalkThroughMainListOnly(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := NewSimpleBlock(chain.genesis.Hash(), myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add 1st chain with 2 blocks after the ancestor
	chain1 := makeBlocks(2, ancestor.Hash(), myNode)
	for i, child := range(chain1) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("1st chain failed to add block #%d: %s", i, err)
		}
	}
	// now add 2nd chain with 1 blocks after the ancestor
	chain2 := makeBlocks(1, ancestor.Hash(), myNode)
	for i, child := range(chain2) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("2nd chain failed to add block #%d: %s", i, err)
		}
	}
	// ask for list of blocks starting at the ancestor
	// should be ancestor, followed by chain1 nodes (3 nodes)
	blocks := chain.Blocks(ancestor.Hash(), 10)
	if len(blocks) != (len(chain1)+1) {
		t.Errorf("chain traversal did not return correct number of blocks: Expected '%d' Found '%d'", 3, len(blocks))
	}
	expected := []*SimpleBlock{ancestor}
	for i, block := range(append(expected, chain1...)) {
		if blocks[i] != block {
			t.Errorf("block at position [%d] is incorrect", i)
		}
	}
}

func TestBlockChainInMemWalkThroughOverMax(t *testing.T) {
	log.SetLogLevel(log.NONE)
	chain := NewBlockChainInMem(testGenesisBlock())
	myNode := NewSimpleNodeInfo("test node")
	// now add a chain with more than max node
	blocks := makeBlocks(maxBlocks+1, chain.genesis.Hash(), myNode)
	for i, child := range(blocks) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("chain failed to add block #%d: %s", i, err)
		}
	}
	// ask for list of blocks more than maxBlocks limit
	syncBlocks := chain.Blocks(chain.genesis.Hash(), maxBlocks+10)
	if len(syncBlocks) > maxBlocks {
		t.Errorf("chain traversal did not return correct number of blocks: Expected '%d' Found '%d'", maxBlocks, len(syncBlocks))
	}
}


func TestBlockChainInMemConsensus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	// simulate 3 different concurrent nodes updating their individual blockchain instances
	chain1, chain2, chain3 := NewBlockChainInMem(testGenesisBlock()), NewBlockChainInMem(testGenesisBlock()), NewBlockChainInMem(testGenesisBlock())
	*chain2.genesis.hash = *chain1.genesis.hash
	chain2.nodes[*chain1.genesis.Hash()] = chain2.genesis
	*chain3.genesis.hash = *chain1.genesis.hash 
	chain3.nodes[*chain1.genesis.Hash()] = chain3.genesis
	node1, node2, node3 := NewSimpleNodeInfo("test node 1"), NewSimpleNodeInfo("test node 2"), NewSimpleNodeInfo("test node 3")
	// define a node function that adds blocks to chain
	lock := sync.RWMutex{}
	counter := 0
	nodeFunc := func(myChain *BlockChainInMem, myNode *SimpleNodeInfo) {
		// simulate mining delay
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(200)))
		// get lock for exclusive access
		lock.RLock()
		defer lock.RUnlock()
		// create a new node using the tip of this node's blockchain
		block := NewSimpleBlock(myChain.Tip().Hash(), myNode)
		block.ComputeHash()
		// simulate broadcast to all nodes
		chain1.AddBlockNode(block)
		chain2.AddBlockNode(block)
		chain3.AddBlockNode(block)
		counter++
	}
	
	// run the node functions on 3 nodes concurrently
	for i := 0; i < 10; i++ {
		go nodeFunc(chain1, node1)
		go nodeFunc(chain2, node2)
		go nodeFunc(chain3, node3)
	}
	// wait for all 3 nodes to finish
	for counter < 30 {}

	// validate that all 3 chains have same tip node hash
	if chain1.Tip().Hash() != chain2.Tip().Hash() {
		t.Errorf("tip of chain1 and chain2 are different")
	}
	if chain2.Tip().Hash() != chain3.Tip().Hash() {
		t.Errorf("tip of chain2 and chain3 are different")
	}
	// validate that all 3 chains have same depth of main/longest chain
	if chain1.Depth() != chain2.Depth() {
		t.Errorf("Depth of chain1 '%d' not same as chain2 '%d'", chain1.Depth(), chain2.Depth())
	}
	if chain2.Depth() != chain3.Depth() {
		t.Errorf("Depth of chain2 '%d' not same as chain3 '%d'", chain2.Depth(), chain3.Depth())
	}
	// validate that all 3 chains have same TD
	if *chain1.TD() != *chain2.TD() {
		t.Errorf("TD of chain1 '%d' not same as chain2 '%d'", *chain1.TD(), *chain2.TD())
	}
	if *chain2.TD() != *chain3.TD() {
		t.Errorf("TD of chain2 '%d' not same as chain3 '%d'", chain2.TD(), chain3.TD())
	}
}

