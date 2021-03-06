package network

import (
    "testing"
    "time"
    "fmt"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/trust-net/go-trust-net/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/go-trust-net/common"
)

func TestNewPlatformManagerNullArgs(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	if _, err := NewPlatformManager(nil, testServiceConfig(), db); err == nil {
		t.Errorf("did not detect nil app config")
	}
	srvConf := testServiceConfig()
	srvConf.TxProcessor = nil
	if _, err := NewPlatformManager(nil, srvConf, db); err == nil {
		t.Errorf("did not detect missing mandatory callback TxProcessor")
	}
	srvConf = testServiceConfig()
	srvConf.PeerValidator = nil
	if _, err := NewPlatformManager(nil, srvConf, db); err == nil {
		t.Errorf("did not detect missing mandatory callback PeerValidator")
	}
	if _, err := NewPlatformManager(nil, testServiceConfig(), db); err == nil {
		t.Errorf("did not detect nil app config")
	}
	if _, err := NewPlatformManager(&AppConfig{}, nil, db); err == nil {
		t.Errorf("did not detect nil service config")
	}
	if _, err := NewPlatformManager(&AppConfig{}, testServiceConfig(), nil); err == nil {
		t.Errorf("did not detect nil app DB")
	}	
}

func TestNewPlatformManagerInterface(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	var mgr PlatformManager
	var err error
	if mgr, err = NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	}
	if err = mgr.Start(); err != nil {
		t.Errorf("Failed to start platform manager: %s", err)
	}
	// submit transaction
	txPayload := []byte("test tx payload")
	txSubmitter := ([]byte("test rx submitter"))
	txSignature := []byte("test rx signature")
	txId := mgr.Submit(txPayload, txSignature, txSubmitter)
	// sleep for some time, for transaction to be processed
	time.Sleep(100 * time.Millisecond)
	if _, err = mgr.Status(txId); err != nil {
		t.Errorf("Failed to get submitted transaction status: %s", err)
	}
	if err = mgr.Stop(); err != nil {
		t.Errorf("Failed to stop platform manager: %s", err)
	}
}

func TestNewPlatformManagerGoodArgs(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		if mgr.Trustee() == nil {
			t.Errorf("platform did not initialize trustee")
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers from new platform instance")
		}
	}
}

func TestValidateAndAddIncorrectNetwork(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		handshake.NetworkId = *core.BytesToByte16([]byte("an invalid network id"))
		peer := &mockNode{}
		if err := mgr.validateAndAdd(handshake, peer); err == nil {
			t.Errorf("Failed to detect network mistmatch in peer handshake")
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers for incorrect handshake")
		}
	}
}

func TestValidateAndAddIncorrectGenesis(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		handshake.Genesis = *core.BytesToByte64([]byte("an invalid genesis"))
		peer := &mockNode{}
		if err := mgr.validateAndAdd(handshake, peer); err == nil {
			t.Errorf("Failed to detect genesis mistmatch in peer handshake")
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers for incorrect handshake")
		}
	}
}

func TestValidateAndAddValidatorError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that will reject peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return core.NewCoreError(0x11, "test error")
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
		}
		if err := mgr.validateAndAdd(handshake, peer); err == nil || err.(*core.CoreError).Code() != 0x11 || err.Error() != "test error" {
			t.Errorf("Failed to detect application validation error in peer handshake: %s", err)
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers for incorrect handshake")
		}
	}
}

func TestValidateAndAddPeerSetUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
		}
		if err := mgr.validateAndAdd(handshake, peer); err != nil {
			t.Errorf("Failed to validate peer handshake: %s", err)
		}
		if mgr.PeerCount() != 1 {
			t.Errorf("did not update peer count")
		}
		if mgr.peerDb.PeerNodeForId(peer.Id()) != peer {
			t.Errorf("did not find peer in peer set DB")
		}
	}
}

func TestUnregisterPeer(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
		}
		if err := mgr.validateAndAdd(handshake, peer); err != nil {
			t.Errorf("Failed to validate peer handshake: %s", err)
		}
		// now disconnect peer
		mgr.unregisterPeer(peer)
		if mgr.PeerCount() != 0 {
			t.Errorf("did not decrement peer count")
		}
		if mgr.peerDb.PeerNodeForId(peer.Id()) != nil {
			t.Errorf("remove peer from peer set DB")
		}
	}
}

func TestHandshakeMsgSendErr(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				sendErr: core.NewCoreError(0x2100000, "test send error"),
			},
		}
		if err := mgr.handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != ErrorHandshakeFailed {
			t.Errorf("Failed to detect handshake msg send error: %s", err)
			return
		}
	}
}

func TestHandshakeMsgReadErr(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				readErr: core.NewCoreError(0x2200000, "test read error"),
				sendSucc: true,
			},
		}
		if err := mgr.handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != 0x2200000 {
			t.Errorf("Failed to detect handshake msg read error: %s", err)
			return
		}
	}
}

func TestHandshakeWrongMsg(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				readResp: p2p.Msg{
					Payload: &testReader{
						response: []byte("some test response"),
					},
					Code: 0x22222,
				},
				sendSucc: true,
			},
		}
		if err := mgr.handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != ErrorHandshakeFailed {
			t.Errorf("Failed to detect incorrect message code: %s", err)
			return
		}
	}
}

func TestHandshakeDecodeError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				readResp: p2p.Msg{
					Payload: &testReader{
						response: nil,
						err: core.NewCoreError(0x232323, "test decode error"),
					},
					Code: Handshake,
				},
				sendSucc: true,
			},
		}
		if err := mgr.handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != ErrorHandshakeFailed {
			t.Errorf("Failed to detect message decode error: %s", err)
			return
		}
	}
}

func TestPlatformManagerSubmitTx(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	var payload []byte
	var block consensus.Block
	conf := testNetworkConfig(func(tx *Transaction) bool{
			called = true
			payload = append(payload, tx.Payload()...)
			block = tx.block
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// submit transaction
		txPayload := []byte("test tx payload")
		txSubmitter := ([]byte("test rx submitter"))
		txSignature := []byte("test rx signature")
		mgr.Submit(txPayload, txSignature, txSubmitter)
		// hack, call to processTx should actually be from block producer go routine
		mgr.processTx(<- mgr.txQ, mgr.engine.NewCandidateBlock())
		if !called {
			t.Errorf("transaction never got processed")
		}
		if string(payload) != "test tx payload" {
			t.Errorf("transaction payload incorrect: '%s'", payload)
		}
		if block == nil {
			t.Errorf("transaction block is nil")
		}
	}
}

func TestPlatformManagerBlockProducer(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	var payload []byte
	var block consensus.Block
	conf := testNetworkConfig(func(tx *Transaction) bool{
			called = true
			payload = append(payload, tx.Payload()...)
			block = tx.block
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		addValidBlock(mgr)

		if !called {
			t.Errorf("transaction never got processed")
		}
		if string(payload) != "test tx payload" {
			t.Errorf("transaction payload incorrect: '%s'", payload)
		}
		if block == nil {
			t.Errorf("transaction block is nil")
		}
	}
}

func TestBlockProducerWaitTimeoutNoGreedyBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	conf := testNetworkConfig(func(tx *Transaction) bool{
			called = true
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// get current tip from consensus engine
		pre := mgr.engine.BestBlock()
		// override timeout
		maxTxWaitSec = 100 * time.Millisecond
		// start block producer
		go mgr.blockProducer()
		// DO NOT submit transaction
		// sleep a bit longer than timeout, no new block should be produced by then
		time.Sleep(150 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		// get tip after timeout
		post := mgr.engine.BestBlock()
		if post.Depth().Uint64() != pre.Depth().Uint64() {
			t.Errorf("a new greedy blocks produced on timeout")
		}
	}
}

func TestPlatformManagerBlockProducerMultipleTransactions(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	callCount := 0
	conf := testNetworkConfig(func(tx *Transaction) bool{
			callCount++
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// get current tip from consensus engine
		pre := mgr.engine.BestBlock()
		// override timeout
		maxTxWaitSec = 100 * time.Millisecond
		// start block producer
		go mgr.blockProducer()
		// submit multiple transactions transaction
		for i := 0; i<5; i++ {
			txPayload := []byte("test tx payload")
			txSubmitter := ([]byte("test rx submitter"))
			txSignature := []byte("test rx signature")
			mgr.Submit(txPayload, txSignature, txSubmitter)
		}
		// sleep a bit, one block should be produced by then
		time.Sleep(100 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		// get tip after timeout
		post := mgr.engine.BestBlock()
		if callCount != 5 {
			t.Errorf("number of processed transactions no correct: %d", callCount)
		}
		if post.Depth().Uint64() != pre.Depth().Uint64()+1 {
			t.Errorf("more than 1 blocks produced")
		}
	}
}

func TestPlatformManagerBlockProducerMaxTransactionsPerBlock(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	callCount := 0
	conf := testNetworkConfig(func(tx *Transaction) bool{
			callCount++
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// get current tip from consensus engine
		pre := mgr.engine.BestBlock()
		// override timeout
		maxTxWaitSec = 100 * time.Millisecond
		// start block producer
		go mgr.blockProducer()
		// submit multiple transactions transaction
		for i := 0; i<15; i++ {
			txPayload := []byte("test tx payload")
			txSubmitter := ([]byte("test rx submitter"))
			txSignature := []byte("test rx signature")
			mgr.Submit(txPayload, txSignature, txSubmitter)
		}
		// sleep a bit, two block should be produced by then
		time.Sleep(100 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		// get tip after timeout
		post := mgr.engine.BestBlock()
		if callCount != 15 {
			t.Errorf("number of processed transactions no correct: %d", callCount)
		}
		if post.Depth().Uint64() != pre.Depth().Uint64()+2 {
			t.Errorf("did not produce 2 blocks")
		}
	}
}

func TestPlatformManagerBlockProducerPowCallback(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	var block consensus.Block
	var hash core.Byte64
	conf := testNetworkConfig(func(tx *Transaction) bool{
			block = tx.block
			return true
		}, func(powHash []byte, ts, delta uint64) bool {
			called = true
			hash = *core.BytesToByte64(powHash)
			return true
		}, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		addValidBlock(mgr)

		// query status of the transaction
		if !called {
			t.Errorf("pow approver never got called")
		}
		if hash != *block.Hash() {
			t.Errorf("pow did not get block's hash")
		}
	}
}

func addValidBlock(mgr *platformManager) {
	// override timeout
	maxTxWaitSec = 100 * time.Millisecond
	// start block producer
	go mgr.blockProducer()
	// submit transaction
	txPayload := []byte("test tx payload")
	txSubmitter := ([]byte("test rx submitter"))
	txSignature := []byte("test rx signature")
	mgr.Submit(txPayload, txSignature, txSubmitter)
	// sleep a bit, hoping transaction will get processed till then
	time.Sleep(100 * time.Millisecond)
	mgr.shutdownBlockProducer <- true
}

func TestPlatformManagerProcessBlockPowCallback(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := 0
	conf := testNetworkConfig(func(tx *Transaction) bool{
			return true
		}, func(powHash []byte, ts, delta uint64) bool {
			called += 1
			log.AppLogger().Debug("PoW Approver called %d times", called)
			return true
		}, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		addValidBlock(mgr)

		// create a copy of current tip (to simulate a network block)
		tip := mgr.engine.(*consensus.BlockChainConsensus).Tip()
		data, _ := common.Serialize(tip.Spec())
		block, _ := mgr.engine.DeserializeNetworkBlock(data)

		// process block (we expect duplicate block error, since its a copy of tip)
		if err := mgr.processBlock(block, nil); err != nil && err.(*core.CoreError).Code() != consensus.ERR_DUPLICATE_BLOCK {
			t.Errorf("Failed to process block: %s", err)
		}
		// check if PoW Approver was called when processing block
		if called < 2 {
			t.Errorf("pow approver only called %d times", called)
		}
	}
}

func testPlatformManagerPowTimeout(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	var block consensus.Block
	var hash *core.Byte64
	conf := testNetworkConfig(func(tx *Transaction) bool{
			block = tx.block
			return true
		}, func(powHash []byte, ts, delta uint64) bool {
			called = true
			hash = core.BytesToByte64(powHash)
			return false
		}, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// start block producer
		go mgr.blockProducer()
		// submit transaction
		txPayload := []byte("test tx payload")
		txSubmitter := []byte("test rx submitter")
		txSignature := []byte("test rx signature")
		txId := mgr.Submit(txPayload, txSignature, txSubmitter)
		finished := false
		go func() {
			fmt.Printf("Waiting for timeout .")
			for !finished {
				// sleep a bit, hoping transaction will get processed till then
				time.Sleep(1000 * time.Millisecond)
				fmt.Printf(".")
			}
			fmt.Printf("done!\n")
		}()
		time.Sleep(1000 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		finished = true
		// query status of the transaction
		if txBlock, err := mgr.Status(txId); err == nil || err.(*core.CoreError).Code() != consensus.ERR_TX_NOT_APPLIED {
			t.Errorf("Failed to detect rejected transaction")
		} else if txBlock != nil {
			t.Errorf("rejected transaction should have nil block")
		}
		// query status of the transaction
		if !called {
			t.Errorf("pow approver never got called")
		}
		if hash == nil {
			t.Errorf("pow did not get block's hash")
		}
	}
}

func TestPlatformManagerPowSuspend(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	var block consensus.Block
	var hash core.Byte64
	var mgr PlatformManager
	var err error
	conf := testNetworkConfig(func(tx *Transaction) bool{
			block = tx.block
			return true
		}, func(powHash []byte, ts, delta uint64) bool {
			called = true
			hash = *core.BytesToByte64(powHash)
			return true
		}, nil)
	if mgr, err = NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	}
	if err = mgr.Start(); err != nil {
		t.Errorf("Failed to start platform manager: %s", err)
	}
	// submit transaction
	txPayload := []byte("test tx payload #1")
	txSubmitter := ([]byte("test rx submitter #2"))
	txId := mgr.Submit(txPayload, nil, txSubmitter)
	// sleep for some time, for transaction to be processed
	time.Sleep(100 * time.Millisecond)
	if _, err = mgr.Status(txId); err != nil {
		t.Errorf("Failed to get submitted transaction status: %s", err)
	}
	// suspend block producer
	if err = mgr.Suspend(); err != nil {
		t.Errorf("Failed to suspend platform manager: %s", err)
	}
	// submit transaction
	txPayload = []byte("test tx payload #1")
	txSubmitter = ([]byte("test rx submitter #2"))
	txId = mgr.Submit(txPayload, nil, txSubmitter)
	// sleep for some time, for transaction to be processed
	time.Sleep(100 * time.Millisecond)
	if _, err = mgr.Status(txId); err == nil || err.(*core.CoreError).Code() != consensus.ERR_TX_NOT_FOUND {
		t.Errorf("did not expect transaction status: %s", err)
	}
	// resume block producer
	if err = mgr.Start(); err != nil {
		t.Errorf("Failed to start platform manager: %s", err)
	}
	// sleep for some time, for transaction to be processed
	time.Sleep(100 * time.Millisecond)
	if _, err = mgr.Status(txId); err != nil {
		t.Errorf("Failed to get submitted transaction status: %s", err)
	}
	
	if err = mgr.Stop(); err != nil {
		t.Errorf("Failed to stop platform manager: %s", err)
	}
}

func TestProcessGreedyBlockValidation(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a greedy block
		block := mgr.engine.NewCandidateBlock()
		data, _ := common.Serialize(block.Spec())
		block, _ = mgr.engine.DeserializeNetworkBlock(data)
		block.AddTransaction(mgr.trustee.NewMiningRewardTx(block))
		
		// process block
		if err := mgr.processBlock(block, nil); err == nil || err.(*core.CoreError).Code() != consensus.ERR_GREEDY_BLOCK {
			t.Errorf("Failed to detect greedy block: %s", err)
		}
	}
}

func TestProcessMiningRewardUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create two new blocks (one is candidate, another is network)
		candidate := mgr.engine.NewCandidateBlock()
		data, _ := common.Serialize(candidate.Spec())
		block, _ := mgr.engine.DeserializeNetworkBlock(data)
		// add mining reward from candidate block into network block
		// (we are doing this so that state of network block does not get updated by reward trustee)
		block.AddTransaction(mgr.trustee.NewMiningRewardTx(candidate))
		txPayload := []byte("test tx payload")
		txSubmitter := []byte("test rx submitter")
		txSignature := []byte("test rx signature")
		block.AddTransaction(consensus.NewTransaction(txPayload, txSignature, txSubmitter))
		// process block
		// (we expect processing to fail because consensus engine will reject block, it did not go through real mining to have correct world state)
		if err := mgr.processBlock(block, NewPeerNode(nil, nil)); err == nil || err.(*core.CoreError).Code() != consensus.ERR_STATE_INCORRECT {
			t.Errorf("Failed to process block: %s", err)
		}
		// get reward balance
		// (we are using block's state, instead of world state, because in our test setup this block is rejected and world state is never updated)
		balance := mgr.trustee.MiningRewardBalance(block, candidate.Miner()).Uint64()
		if  balance != 1000000 {
			t.Errorf("Failed to award mining reward, balance: %d", balance)
		}
	}
}

func TestProcessUncleRewardUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		txPayload := []byte("test tx payload")
		txSubmitter := []byte("test rx submitter")
		txSignature := []byte("test rx signature")
		// build a forked chain
		uncle := mgr.engine.NewCandidateBlock()
		parent := mgr.engine.NewCandidateBlock()
		uncle.AddTransaction(mgr.trustee.NewMiningRewardTx(uncle))
		uncle.AddTransaction(consensus.NewTransaction(txPayload, txSignature, txSubmitter))
		if !mgr.mineCandidateBlock(uncle) {
			t.Errorf("Failed to process uncle block: %s", err)
		}
		parent.AddTransaction(mgr.trustee.NewMiningRewardTx(parent))
		parent.AddTransaction(consensus.NewTransaction(txPayload, txSignature, txSubmitter))
		if !mgr.mineCandidateBlock(parent) {
			t.Errorf("Failed to process parent block: %s", err)
		}

		// create a new child block from network by serialize/de-serialize a new candidate block
		childDiscard := mgr.engine.NewCandidateBlock()
		data, _ := common.Serialize(childDiscard.Spec())
		child, _ := mgr.engine.DeserializeNetworkBlock(data)

		// add mining reward from candidate block into network block
		child.AddTransaction(mgr.trustee.NewMiningRewardTx(childDiscard))
		child.AddTransaction(consensus.NewTransaction(txPayload, txSignature, txSubmitter))

		// process block
		if err := mgr.processBlock(child, NewPeerNode(nil, nil)); err == nil || err.(*core.CoreError).Code() != consensus.ERR_STATE_INCORRECT {
			t.Errorf("Failed to process child block: %s", err)
		}

		// get reward balance
		balance := mgr.trustee.MiningRewardBalance(child, childDiscard.Miner()).Uint64()
		if  balance != 200000 + 1000000 + 1000000 {
			t.Errorf("incorrect mining reward with uncle: %d", balance)
		}
	}
}
