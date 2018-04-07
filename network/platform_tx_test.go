package network

import (
    "testing"
    "time"
//    "fmt"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/core"
//	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/consensus"
)

func TestPlatformManagerTransactionStatus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	var block consensus.Block
	conf := testNetworkConfig(func(tx *Transaction) bool{
			block = tx.block
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// override timeout
		maxTxWaitSec = 100 * time.Millisecond
		// start block producer
		go mgr.blockProducer()
		// submit transaction
		txPayload := []byte("test tx payload")
		txSubmitter := core.BytesToByte64([]byte("test rx submitter"))
		txSignature := core.BytesToByte64([]byte("test rx signature"))
		txId := mgr.Submit(txPayload, txSubmitter, txSignature)
		// sleep a bit, hoping transaction will get processed till then
		time.Sleep(100 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		// query status of the transaction
		if txBlock, err := mgr.Status(txId); err != nil {
			t.Errorf("Failed to get transaction status: %s", err)
		} else if *txBlock.Hash() != *block.Hash() {
			t.Errorf("submitted transaction's block incorrect: %s", err)
		}
	}
}

func TestPlatformManagerUnknownTransactionStatus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(func(tx *Transaction) bool{
			return true
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// override timeout
		maxTxWaitSec = 100 * time.Millisecond
		// start block producer
		go mgr.blockProducer()
		// submit transaction
		txPayload := []byte("test tx payload")
		txSubmitter := core.BytesToByte64([]byte("test rx submitter"))
		txSignature := core.BytesToByte64([]byte("test rx signature"))
		mgr.Submit(txPayload, txSubmitter, txSignature)
		txId := core.BytesToByte64([]byte("some random transaction Id"))
		// sleep a bit, hoping transaction will get processed till then
		time.Sleep(100 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		// query status of the transaction
		if _, err := mgr.Status(txId); err == nil || err.(*core.CoreError).Code() != consensus.ERR_TX_NOT_FOUND {
			t.Errorf("Failed to detect unknown transaction")
		}
	}
}

func TestPlatformManagerRejectedTransactionStatus(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(func(tx *Transaction) bool{
			return false
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// override timeout
		maxTxWaitSec = 100 * time.Millisecond
		// start block producer
		go mgr.blockProducer()
		// submit transaction
		txPayload := []byte("test tx payload")
		txSubmitter := core.BytesToByte64([]byte("test rx submitter"))
		txSignature := core.BytesToByte64([]byte("test rx signature"))
		txId := mgr.Submit(txPayload, txSubmitter, txSignature)
		// sleep a bit, hoping transaction will get processed till then
		time.Sleep(100 * time.Millisecond)
		mgr.shutdownBlockProducer <- true
		// query status of the transaction
		if txBlock, err := mgr.Status(txId); err == nil || err.(*core.CoreError).Code() != consensus.ERR_TX_NOT_APPLIED {
			t.Errorf("Failed to detect rejected transaction")
		} else if txBlock != nil {
			t.Errorf("rejected transaction should have nil block")
		}
	}
}

func TestPlatformManagerValidateTx(t *testing.T) {
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
		// create a transaction
		txPayload := []byte("test tx payload")
		txSubmitter := core.BytesToByte64([]byte("test rx submitter"))
		txSignature := core.BytesToByte64([]byte("test rx signature"))
		txBlock := mgr.engine.NewCandidateBlock()
		tx := consensus.NewTransaction(txPayload, txSubmitter, txSignature)
		result := mgr.processTx(tx, txBlock)
		if !called {
			t.Errorf("callback did not get called")
		}
		if !result {
			t.Errorf("Failed to process transaction")
		}
	}
}
