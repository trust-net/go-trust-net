package network

import (
    "testing"
    "time"
    "crypto/ecdsa"
    "crypto/rand"
    "math/big"
//	"crypto/sha512"
    "fmt"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/ethereum/go-ethereum/crypto"
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
		txSubmitter := []byte("test rx submitter")
		txSignature := []byte("test rx signature")
		txId := mgr.Submit(txPayload, txSignature, txSubmitter)
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
		txSubmitter := []byte("test rx submitter")
		txSignature := []byte("test rx signature")
		mgr.Submit(txPayload, txSignature, txSubmitter)
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
		txSubmitter := []byte("test rx submitter")
		txSignature := []byte("test rx signature")
		txId := mgr.Submit(txPayload, txSignature, txSubmitter)
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

func TestPlatformManagerValideTxSignature(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	type signature struct {
		R *big.Int
		S *big.Int
	}
	conf := testNetworkConfig(func(tx *Transaction) bool{
			called = true
			// extract submitter's key
			key := crypto.ToECDSAPub(tx.Submitter())
			if key == nil {
				return false
			}
			// validate signature of payload
			s := signature{}
			if err := common.Deserialize(tx.Signature(), &s); err != nil {
				fmt.Printf("Failed to parse signature: %s\n", err)
				return false
			}
			validation := ecdsa.Verify(key, tx.Payload(), s.R, s.S)
			if !validation {
				fmt.Printf("Failed to validate signature: %x\n", tx.Signature()) 
			}
			return validation
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a transaction
		txPayload := []byte("test tx payload with different value")
		// build a submitter key pair
		key, _ := crypto.GenerateKey()
		idBytes := crypto.FromECDSAPub(&key.PublicKey)
		txSubmitter := idBytes
		s := signature{}
		var err error
		if s.R,s.S, err = ecdsa.Sign(rand.Reader, key, txPayload); err != nil {
			t.Errorf("Failed to sign transaction: %s", err)
		}
		var txSignature []byte
		if txSignature,err = common.Serialize(s); err != nil {
			t.Errorf("Failed to serialize signature: %s", err)
			return
		}
//		fmt.Printf("Signature length %d : %x\n", len(txSignature), txSignature)
		txBlock := mgr.engine.NewCandidateBlock()
		tx := consensus.NewTransaction(txPayload, txSignature, txSubmitter)
		result := mgr.processTx(tx, txBlock)
		if !called {
			t.Errorf("callback did not get called")
		}
		if !result {
			t.Errorf("Failed to process transaction")
		}
	}
}

func TestPlatformManagerInvalidTxSignature(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	called := false
	type signature struct {
		R *big.Int
		S *big.Int
	}
	conf := testNetworkConfig(func(tx *Transaction) bool{
			called = true
			// extract submitter's key
			key := crypto.ToECDSAPub(tx.Submitter())
			if key == nil {
				return false
			}
			// validate signature of payload
			s := signature{}
			if err := common.Deserialize(tx.Signature(), &s); err != nil {
				fmt.Printf("Failed to parse signature: %s\n", err)
				return false
			}
//			fmt.Printf("Reciever Signature: %s\n", s)
			validation := ecdsa.Verify(key, tx.Payload(), s.R, s.S)
//			if !validation {
//				fmt.Printf("Failed to validate signature: %x\n", tx.Signature()) 
//			}
			return validation
		}, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a transaction
		txPayload := []byte("test tx payload with different value")
		// build two submitter key pairs
		submitter1, _ := crypto.GenerateKey()
		submitter2, _ := crypto.GenerateKey()
		
		// use first submitter's identity to submit transaction
		idBytes := crypto.FromECDSAPub(&submitter1.PublicKey)
//		fmt.Printf("idBytes length %d : %x\n", len(idBytes), idBytes)
		txSubmitter := idBytes
//		fmt.Printf("submitter length %d : %x\n", len(txSubmitter), txSubmitter)

		// use second submitter's keys to sign transaction
		s := signature{}
		var err error
		if s.R,s.S, err = ecdsa.Sign(rand.Reader, submitter2, txPayload); err != nil {
			t.Errorf("Failed to sign transaction: %s", err)
		}
//		fmt.Printf("Sender Signature: %s\n", s)
		var txSignature []byte
		if txSignature,err = common.Serialize(s); err != nil {
			t.Errorf("Failed to serialize signature: %s", err)
			return
		}

		txBlock := mgr.engine.NewCandidateBlock()
		tx := consensus.NewTransaction(txPayload, txSignature, txSubmitter)
		result := mgr.processTx(tx, txBlock)
		if !called {
			t.Errorf("callback did not get called")
		}
		if result {
			t.Errorf("Failed to validate transaction signature")
		}
	}
}
