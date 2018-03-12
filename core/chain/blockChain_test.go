package chain

import (
    "testing"
    "time"
    "fmt"
    "math/rand"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/db"
)

func testGenesisBlock(ts uint64) core.Block{
	b := core.NewSimpleBlock(core.BytesToByte64(nil), 0, 0, ts, core.NewSimpleNodeInfo(""))
	b.ComputeHash()
	return b
}

func TestBlockNodeEncode(t *testing.T) {
	now := uint64(time.Now().UnixNano())
	genesis := testGenesisBlock(now)
	node := NewBlockNode(genesis)
	if data, err := common.Serialize(node); err != nil {
		t.Errorf("failed to encode block node: %s", err)
	} else {
		fmt.Printf("Encoded: %s\n", data[:12])
	}
}

func TestBlockNodeDecode(t *testing.T) {
	now := uint64(time.Now().UnixNano())
	block := testGenesisBlock(now)
	data, _ := common.Serialize(NewBlockNode(block))
	var node BlockNode
	if err := common.Deserialize(data, &node); err != nil {
		t.Errorf("failed to decode block node: %s", err)
	} else {
		fmt.Printf("Decoded: %d\n", node.Depth())
	}
}

func TestNewBlockChainInMem(t *testing.T) {
	log.SetLogLevel(log.NONE)
	now := uint64(time.Now().UnixNano())
	db, _ := db.NewDatabaseInMem()
	chain, err := NewBlockChainInMem(testGenesisBlock(now), db)
	if err != nil {
		t.Errorf("failed to create block chain: %s", err)
		return
	}
	if chain.depth != 0 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 0, chain.depth)
	}
	if chain.genesis.Depth() != 0 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 0, chain.depth)
	}
	if now != chain.td.Uint64() {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", now, chain.TD())
	}
	// verify in mem db
//	for k,v := range db.Map() {
//		fmt.Printf("Key: '%s', Value: '%s'\n", k,v)
//	}
//	if data, err := db.Get(tableKey(tableBlockNode, chain.genesis.Hash())); err != nil {
//		t.Errorf("error checking genesis in DB: %s, '%s'", err, data)
//		return
//	} else if len(data) <= 0 {
//		t.Errorf("genesis not in DB")
//		return
//	}
	bn, found := chain.BlockNode(chain.genesis.Hash())
	if !found {
		t.Errorf("did not find genesis node")
		return
	}
	if !bn.IsMainList() {
		t.Errorf("did not set main list flag on genesis node")
	}
	if *chain.Tip().Hash() != *bn.Hash() {
		t.Errorf("did not set block chain tip to genesis node. Expected '%x', found '%x'", bn.Hash(), chain.Tip().Hash())
	}
	if chain.Tip().Depth() != 0 {
		t.Errorf("did not set depth of genesis node as 0")
	}
}

func TestBlockChainInMemAddNode(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	block.ComputeHash()
	if err := chain.AddBlockNode(block); err != nil {
		t.Errorf("Failed to add block into chain: '%s'", err)
	}
	bn,_ := chain.BlockNode(block.Hash())
	if !bn.IsMainList() {
		t.Errorf("did not update main list flag on new node")
	}
	if *bn.Parent() != *chain.genesis.Hash() {
		t.Errorf("did not set parent link on new node")
	}
}

func TestBlockChainInMemAddNodeGenesis(t *testing.T) {
	log.SetLogLevel(log.NONE)
	genesis := testGenesisBlock(0x20000)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(genesis, db)
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if *chain.Genesis().Hash() != *genesis.Hash() {
		t.Errorf("Genesis reference not correct")
	}
}

func TestBlockChainInMemAddNodeDepthUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if chain.Depth() != 1 {
		t.Errorf("Failed to update depth of the chain")
	}
}

func TestBlockChainInMemAddNodeTdUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	if *chain.TD() != *block.Timestamp() {
		t.Errorf("Failed to update TD of the chain: Expected %d, Actual %d", block.Timestamp(), chain.TD())
	}
}

func TestBlockChainFlush(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	genesis := testGenesisBlock(0x20000)
	chain, _ := NewBlockChainInMem(genesis, db)
	myNode := core.NewSimpleNodeInfo("test node")
	// now add descendents with 6 blocks after the genesis
	list := makeBlocks(6, genesis.Hash(), myNode, 1, 1)
	for i, child := range(list) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("chain failed to add block #%d: %s", i, err)
		}
	}
	// reset/flush the chain
	if err := chain.Flush(); err != nil {
		t.Errorf("chain failed to flush: %s", err)
	}
	if *chain.Tip().Hash() != *genesis.Hash() {
		t.Errorf("chain flush did not reset tip: Expected '%x', Found '%x'", genesis.Hash(), chain.Tip())
	}
	if chain.Depth() != 0 {
		t.Errorf("chain flush did not reset depth: Expected '%x', Found '%x'", 0, chain.Depth())
	}
	if *chain.TD() != *genesis.Timestamp() {
		t.Errorf("chain flush did not reset TD: Expected '%x', Found '%x'", genesis.Timestamp(), chain.TD())
	}
	if child := chain.findMainListChild(chain.genesis, nil); child != nil {
		t.Errorf("chain flush did not reset main list: Expected '%x', Found '%x'", nil, child.Hash())
	}
	// verify that all descendends were deleted
	for _, child := range(list) {
		if _, found := chain.BlockNode(child.Hash()); found {
			t.Errorf("chain failed to delete DAG node %x", child.Hash())
		}
		if _, found := chain.Block(child.Hash()); found {
			t.Errorf("chain failed to delete block %x", child.Hash())
		}
	}
}

func TestBlockChainInMemAddNodeUncomputed(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected block with uncomputed hash")
	}
}

func TestBlockChainInMemAddNodeNil(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	if err := chain.AddBlockNode(nil); err == nil {
		t.Errorf("Failed to detected nil block")
	}
}

func TestBlockChainInMemAddNodeOrphan(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	block := core.NewSimpleBlock(core.BytesToByte64([]byte("some random parent hash")), 1, 1, 0, myNode)
	block.ComputeHash()
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected block with non existing parent")
	}
}


func TestBlockChainInMemAddNodeDuplicate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// add a block to chain
	block := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	block.ComputeHash()
	chain.AddBlockNode(block)
	// try adding same block again
	if err := chain.AddBlockNode(block); err == nil {
		t.Errorf("Failed to detected duplicate block")
	}
}

func TestBlockChainInMemAddNodeForwardLink(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add two child nodes to same ancestor
	child1 := core.NewSimpleBlock(ancestor.Hash(), 2, 2, 0, myNode)
	child1.ComputeHash()
	chain.AddBlockNode(child1)
	child2 := core.NewSimpleBlock(ancestor.Hash(), 2, 2, 0, myNode)
	child2.ComputeHash()
	chain.AddBlockNode(child2)
	// verify that ancestor has forward link to children
	parent, _ := chain.BlockNode(ancestor.Hash())
	if len(parent.Children()) != 2 {
		t.Errorf("chain did not update forward links")
	}
	if *parent.Children()[0] != *child1.Hash() {
		t.Errorf("chain added incorrect forward link 1st child: Expected '%d', Found '%d'", child1.Hash(), parent.Children()[0])
	}
	if *parent.Children()[1] != *child2.Hash() {
		t.Errorf("chain added incorrect forward link 2nd child: Expected '%d', Found '%d'", child2.Hash(), parent.Children()[1])
	}
}

func makeBlocks(len int, parent *core.Byte64, miner core.NodeInfo, startWeight, startDepth uint64) []*core.SimpleBlock {
	nodes := make([]*core.SimpleBlock, len)
	for i := uint64(0); i < uint64(len); i++ {
		nodes[i] = core.NewSimpleBlock(parent, startWeight+i, startDepth+i, 0, miner)
		nodes[i].ComputeHash()
		parent = nodes[i].Hash()
	}
	return nodes
}

func addChain(chain *BlockChainInMem, blocks []*core.SimpleBlock, miner core.NodeInfo) error{
	var parent *core.SimpleBlock
	parent = nil  
	for i, block := range(blocks) {
		if parent != nil {
			block = core.NewSimpleBlock(parent.Hash(), parent.Weight().Uint64()+1, parent.Depth().Uint64()+1, 0, miner)
			block.ComputeHash()
		}
		if err := chain.AddBlockNode(block); err != nil {
			return err
		}
		parent = block
		blocks[i] = block
	}
	return nil
}

func TestBlockChainInMemHeaviestChain(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add 1st chain with 6 blocks after the ancestor
	chain1 := makeBlocks(6, ancestor.Hash(), myNode, 2, 2)
	if err := addChain(chain, chain1, myNode); err != nil {
		t.Errorf("1st chain failed to add block: %s", err)
	}
	// now add 2nd chain with 4 blocks after the ancestor
	// each of these new block additions will also include uncle weights from 1st chain
	chain2 := makeBlocks(4, ancestor.Hash(), myNode, 2, 2)
	log.SetLogLevel(log.NONE)
	if err := addChain(chain, chain2, myNode); err != nil {
		t.Errorf("2nd chain failed to add block: %s", err)
	}
	log.SetLogLevel(log.NONE)
	// validate that heaviest chain (chain2, length 1+4) wins
	if chain.Depth() != 5 {
		t.Errorf("chain depth incorrect: Expected '%d' Found '%d'", 5, chain.Depth())
	}
	if *chain.TD() != *chain2[3].Timestamp() {
		t.Errorf("chain TD incorrect: Expected '%d' Found '%d'", chain2[3].Timestamp().Uint64(), chain.TD().Uint64())
	}
}

func TestBlockChainInMemRebalance(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// add 1st child to ancestor
	child1 := core.NewSimpleBlock(ancestor.Hash(), 2, 2, 0, myNode)
	child1.ComputeHash()
	chain.AddBlockNode(child1)
	// add 2nd child to ancestor
	child2 := core.NewSimpleBlock(ancestor.Hash(), 2, 2, 0, myNode)
	child2.ComputeHash()
	chain.AddBlockNode(child2)
	// verify that blockchain points to 1st child as tip
	if *chain.Tip().Hash() != *child1.Hash() {
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
	child3 := core.NewSimpleBlock(child2.Hash(), child2.Weight().Uint64()+1, child2.Depth().Uint64()+1, 0, myNode)
	child3.ComputeHash()
	chain.AddBlockNode(child3)
	// verify that blockchain points to 3rd child as tip
	if *chain.Tip().Hash() != *child3.Hash() {
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
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// add an ancestor block to chain
	ancestor := core.NewSimpleBlock(chain.genesis.Hash(), 1, 1, 0, myNode)
	ancestor.ComputeHash()
	chain.AddBlockNode(ancestor)
	// now add 1st chain with 2 blocks after the ancestor
	chain1 := makeBlocks(2, ancestor.Hash(), myNode, 2, 2)
	for i, child := range(chain1) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("1st chain failed to add block #%d: %s", i, err)
		}
	}
	// now add 2nd chain with 1 blocks after the ancestor
	chain2 := makeBlocks(1, ancestor.Hash(), myNode, 2, 2)
	for i, child := range(chain2) {
		if err := chain.AddBlockNode(child); err != nil {
			t.Errorf("2nd chain failed to add block #%d: %s", i, err)
		}
	}
	// ask for list of blocks starting at the ancestor
	// should skip ancestor qnd return chain1 nodes (2 nodes)
	blocks := chain.Blocks(ancestor.Hash(), 10)
	if len(blocks) != (len(chain1)) {
		t.Errorf("chain traversal did not return correct number of blocks: Expected '%d' Found '%d'", 2, len(blocks))
	}
	for i, block := range(chain1) {
		if *blocks[i].Hash() != *block.Hash() {
			t.Errorf("block at position [%d] is incorrect", i)
		}
	}
}

func TestBlockChainInMemWalkThroughOverMax(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// now add a chain with more than max node
	blocks := makeBlocks(maxBlocks+1, chain.genesis.Hash(), myNode, 1, 1)
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
	log.SetLogLevel(log.DEBUG)
	// simulate 3 different concurrent nodes updating their individual blockchain instances
	db1, _ := db.NewDatabaseInMem()
	chain1, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db1)
	db2, _ := db.NewDatabaseInMem()
	chain2, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db2)
	db3, _ := db.NewDatabaseInMem()
	chain3, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db3)
	node1, node2, node3 := core.NewSimpleNodeInfo("test node 1"), core.NewSimpleNodeInfo("test node 2"), core.NewSimpleNodeInfo("test node 3")
	// define a node function that adds blocks to chain
	counter := 0
	nodeFunc := func(myChain *BlockChainInMem, myNode *core.SimpleNodeInfo) {
		// simulate mining delay
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(200)))
		// get lock for exclusive access
		myChain.lock.Lock()
		// create a new node using the tip of this node's blockchain
		tip := myChain.Tip()
		parent, _ := myChain.BlockNode(tip.Parent())
		myChain.lock.Unlock()
		block := core.NewSimpleBlock(tip.Hash(), tip.Weight()+1, tip.Depth()+1, 0, myNode)
		// for every 7th block, simulate a delayed/heavier block
		if (counter+1) % 7 == 0 {
			block = core.NewSimpleBlock(parent.Hash(), parent.Weight()+1, parent.Depth()+1, 0, myNode)
		}
		block.ComputeHash()
		// simulate broadcast to all nodes
		switch myChain {
			case chain1:
				// add local block
				if err := chain1.AddBlockNode(block); err != nil {
					fmt.Printf("%s failed to add new block to chain1 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
				// broadcast network block
				if err := chain2.AddNetworkNode(core.NewSimpleBlockFromSpec(core.NewBlockSpecFromBlock(block))); err != nil {
					fmt.Printf("%s failed to add network block to chain2 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
				if err := chain3.AddNetworkNode(core.NewSimpleBlockFromSpec(core.NewBlockSpecFromBlock(block))); err != nil {
					fmt.Printf("%s failed to add network block to chain3 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
			case chain2:
				// add local block
				if err := chain2.AddBlockNode(block); err != nil {
					fmt.Printf("%s failed to add new block to chain2 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
				// broadcast network block
				if err := chain1.AddNetworkNode(core.NewSimpleBlockFromSpec(core.NewBlockSpecFromBlock(block))); err != nil {
					fmt.Printf("%s failed to add network block to chain1 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
				if err := chain3.AddNetworkNode(core.NewSimpleBlockFromSpec(core.NewBlockSpecFromBlock(block))); err != nil {
					fmt.Printf("%s failed to add network block to chain3 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
			case chain3:
				// add local block
				if err := chain3.AddBlockNode(block); err != nil {
					fmt.Printf("%s failed to add new block to chain3 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
				// broadcast network block
				if err := chain2.AddNetworkNode(core.NewSimpleBlockFromSpec(core.NewBlockSpecFromBlock(block))); err != nil {
					fmt.Printf("%s failed to add network block to chain2 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
				if err := chain1.AddNetworkNode(core.NewSimpleBlockFromSpec(core.NewBlockSpecFromBlock(block))); err != nil {
					fmt.Printf("%s failed to add network block to chain1 at depth %d: %s\n", myNode.Id(), block.Depth().Uint64(), err)
				}
		}
		counter++
		fmt.Printf("%s : chain depth: %d, chain weight: %d, Counter: %d\n", myNode.Id(), myChain.Depth(), myChain.Weight(), counter)
	}
	
	// run the node functions on 3 nodes concurrently
	for i := 0; i < 10; i++ {
		go nodeFunc(chain1, node1)
		go nodeFunc(chain2, node2)
		go nodeFunc(chain3, node3)
	}
	// wait for all nodes to finish
	for counter < 30 {time.Sleep(time.Millisecond * 100)}
	// validate that all 3 chains have same tip node hash
	if *chain1.Tip().Hash() != *chain2.Tip().Hash() {
		t.Errorf("tip of chain1 and chain2 are different")
	}
	if *chain2.Tip().Hash() != *chain3.Tip().Hash() {
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

func TestBlockChainUncleUpdates(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// build chain 1 with 6 deep
	chain1blocks := makeBlocks(6, chain.genesis.Hash(), myNode, 1, 1)
	if err := addChain(chain, chain1blocks, myNode); err != nil {
		t.Errorf("chain failed to add block: %s", err)
	}
	// add a new block forked immediately at genesis
	block_2_1 := makeBlocks(1, chain.genesis.Hash(), myNode, 1, 1)[0]
	chain.AddBlockNode(block_2_1)
	// verify that this block shows up as uncle for 5th block on main chain
	log.SetLogLevel(log.NONE)
	uncles := chain.findUncles(chain1blocks[2].Hash(), chain1blocks[3].Hash(), maxUncleDistance, chain1blocks[3].Depth().Uint64())
	if len(uncles) != 1 {
		t.Errorf("did not find any uncles for block at depth %d", chain1blocks[4].Depth().Uint64())
		return
	}
	if *uncles[0].hash != *block_2_1.Hash() {
		t.Errorf("Incorrect uncle: Expected %x, found %x", *block_2_1.Hash(), *uncles[0].hash)
	}
	// add a new chain forked at 3rd block in main chain
	chain2blocks := makeBlocks(3, chain1blocks[2].Hash(), myNode, chain1blocks[2].Weight().Uint64()+1, chain1blocks[2].Depth().Uint64()+1)
	log.SetLogLevel(log.NONE)
	if err := addChain(chain, chain2blocks, myNode); err != nil {
		t.Errorf("chain failed to add block: %s", err)
	}
	// verify that forked chain blocks show up as uncles
	log.SetLogLevel(log.NONE)
	uncles = chain.findUncles(chain1blocks[2].Hash(), chain1blocks[3].Hash(), maxUncleDistance, chain1blocks[3].Depth().Uint64())
	// we expect the 1st forked block at genesis and only 1 forked chain block as uncle (others in forked chain have same or greater depth)
	if len(uncles) != 2 {
		t.Errorf("found incorrect number of uncles, expected %d, found %d", 2, len(uncles))
	}
	for _, uncle := range uncles {
		if *uncle.hash == *chain2blocks[1].Hash() || *uncle.hash == *chain2blocks[2].Hash() {
			t.Errorf("unexpected uncle %x at depth %d", *uncle.hash, uncle.depth)
		}
	} 
	log.SetLogLevel(log.NONE)
	uncles = chain.findUncles(chain1blocks[4].Hash(), chain1blocks[5].Hash(), maxUncleDistance, chain1blocks[5].Depth().Uint64())
	// we expect the 3 forked chain blocks as uncle, but not the 1st forked block at genesis
	if len(uncles) != 3 {
		t.Errorf("found incorrect number of uncles, expected %d, found %d", 4, len(uncles))
	}
	for _, uncle := range uncles {
		if *uncle.hash == *block_2_1.Hash() {
			t.Errorf("unexpected uncle %x at depth %d", *uncle.hash, uncle.depth)
		}
	} 
}

func TestBlockChainAddBlockWithUncles(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// build chain 1 with 5 deep
	chain1blocks := makeBlocks(5, chain.genesis.Hash(), myNode, 1, 1)
	for i, block := range(chain1blocks) {
		if err := chain.AddBlockNode(block); err != nil {
			t.Errorf("chain failed to add block #%d: %s", i, err)
		}
	}
	// add a new uncle block to 3rd block in main chain
	uncle := makeBlocks(1, chain1blocks[2].Hash(), myNode, chain1blocks[2].Weight().Uint64()+1, chain1blocks[2].Depth().Uint64()+1)[0]
	chain.AddBlockNode(uncle)

	// now add a new child block to main chain
	log.SetLogLevel(log.NONE)
	child := makeBlocks(1, chain1blocks[4].Hash(), myNode, chain1blocks[4].Weight().Uint64()+1, chain1blocks[4].Depth().Uint64()+1)[0]
	origWeight := child.Weight().Uint64()
	origHash := child.Hash()
	chain.AddBlockNode(child)

	// verify that chain addition recomputed hash
	if *origHash == *child.Hash() {
		t.Errorf("chain did not recompute hash when adding block with uncles")
	}
	// fetch the child back from blockchain
	fetched, _ := chain.Block(child.Hash())
	if fetched == nil {
		t.Errorf("chain failed to fetch block %x", child.Hash())
	}
	if len(fetched.Uncles()) != 1 {
		t.Errorf("chain did not add uncles to the block when adding")
	}
	if fetched.Weight().Uint64() != origWeight+1 {
		t.Errorf("chain did not update weight of the block when adding, expected %d, found %d", origWeight+1, fetched.Weight().Uint64())
	}
}


func TestBlockChainAddNetworkBlockWithUnknownUncle(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// build chain 1 with 5 deep
	chain1blocks := makeBlocks(5, chain.genesis.Hash(), myNode, 1, 1)
	for i, block := range(chain1blocks) {
		if err := chain.AddBlockNode(block); err != nil {
			t.Errorf("chain failed to add block #%d: %s", i, err)
		}
	}
	// prepare a new child block to be added to chain
	log.SetLogLevel(log.NONE)
	child := makeBlocks(1, chain1blocks[4].Hash(), myNode, chain1blocks[4].Weight().Uint64()+1, chain1blocks[4].Depth().Uint64()+1)[0]
	// add a fake uncle block to the new child block
	child.AddUncle(core.BytesToByte64([]byte("fake uncle")), 1)
	child.ComputeHash()
	// attempt to add block to chain, it should fail
	if err := chain.AddNetworkNode(child); err == nil {
		t.Errorf("failed to detect incorrect uncle")
	} else if err.(*core.CoreError).Code() != core.ERR_INVALID_UNCLE {
		t.Errorf("incorrect error code: Expected %d, Found %d", core.ERR_INVALID_UNCLE, err.(*core.CoreError).Code())
	} else {
		fmt.Printf("Detected correct: %s\n", err)
	}
}

func TestBlockChainRebalanceByWeight(t *testing.T) {
	log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	chain, _ := NewBlockChainInMem(testGenesisBlock(0x20000), db)
	myNode := core.NewSimpleNodeInfo("test node")
	// build chain 1 with 5 deep
	chain1blocks := makeBlocks(5, chain.genesis.Hash(), myNode, 1, 1)
	for i, block := range(chain1blocks) {
		if err := chain.AddBlockNode(block); err != nil {
			t.Errorf("chain failed to add block #%d: %s", i, err)
		}
	}
	// now add a new child block to main chain
	log.SetLogLevel(log.NONE)
	child := makeBlocks(1, chain1blocks[4].Hash(), myNode, chain1blocks[4].Weight().Uint64()+1, chain1blocks[4].Depth().Uint64()+1)[0]
	chain.AddBlockNode(child)
	// validate that tip of the blockchain points to this child
	if *chain.Tip().Hash() != *child.Hash() || chain.Tip().Depth() != 6 {
		t.Errorf("chain tip incorrect")
	}

	// now add two new uncle blocks to 3rd block in main chain
	uncle1 := makeBlocks(1, chain1blocks[2].Hash(), myNode, chain1blocks[2].Weight().Uint64()+1, chain1blocks[2].Depth().Uint64()+1)[0]
	uncle2 := makeBlocks(1, chain1blocks[2].Hash(), myNode, chain1blocks[2].Weight().Uint64()+1, chain1blocks[2].Depth().Uint64()+1)[0]
	chain.AddBlockNode(uncle1)
	chain.AddBlockNode(uncle2)
	
	// now add a new child to the 4th block in main chain, which will include above two uncles
	log.SetLogLevel(log.NONE)
	heavierChild := makeBlocks(1, chain1blocks[3].Hash(), myNode, chain1blocks[3].Weight().Uint64()+1, chain1blocks[3].Depth().Uint64()+1)[0]
	chain.AddBlockNode(heavierChild)
	// validate that blockchain has rebalanced and now tip points to heavier child
	if *chain.Tip().Hash() != *heavierChild.Hash() {
		t.Errorf("chain tip incorrect")
	}
	if chain.Tip().Depth() >  child.Depth().Uint64() {
		t.Errorf("chain depth incorrect")
	}
	if chain.Tip().Weight() <  child.Weight().Uint64() {
		t.Errorf("chain weight incorrect")
	}
}
