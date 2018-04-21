package trustee

import (
//	"fmt"
	"strconv"
    "testing"
	"crypto/ecdsa"
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/network"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNewTrusteeAppInterface(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	var trustee Trustee
	trustee = NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
	}
}

func TestTrusteeSignPayload(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
		return
	}
	payload := []byte("this is a random payload")
	// sign the payload
	sign := trustee.sign(payload)
	// desrialize signature
	signature := &signature{}
	if err := common.Deserialize(sign, signature); err != nil {
		t.Errorf("failed to desrialize signature: %s", err)
		return
	}
	// signing should have hashed the payload
	hash := sha512.Sum512(payload)
	// validate signature using public key
	if !ecdsa.Verify(&nodekey.PublicKey, hash[:], signature.R, signature.S) {
		t.Errorf("failed to verify payload signature")
	}
}
//
//func TestTrusteeSignPayloadFailure(t *testing.T) {
//	log.SetLogLevel(log.NONE)
//	defer log.SetLogLevel(log.NONE)
//	mgr := MockPlatformManager{}
//	nodekey, _ := crypto.GenerateKey()
//	config := &network.AppConfig{}
//	trustee := NewTrusteeApp(&mgr, nodekey, config)
//	if trustee == nil {
//		t.Errorf("failed to get trustee instance")
//		return
//	}
//	// sign the payload
//	sign := trustee.sign(nil)
//	if sign != nil {
//		t.Errorf("payload signing did not fail for bad payload")
//	}
//}

func TestTrusteeVerifyPayload(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
		return
	}
	payload := []byte("this is a random payload")
	// validate signed payload with with same trustee instance
	if !trustee.verify(payload, trustee.sign(payload), crypto.FromECDSAPub(&nodekey.PublicKey)) {
		t.Errorf("failed to validate signature using trustee")
	}
}

func TestTrusteeVerifyPayloadWrongKey(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
		return
	}
	payload := []byte("this is a random payload")
	// use a different key
	nodekey2, _ := crypto.GenerateKey()
	// validate signed payload with with same trustee instance but different key in transaction
	if trustee.verify(payload, trustee.sign(payload), crypto.FromECDSAPub(&nodekey2.PublicKey)) {
		t.Errorf("signature verification did not detect wrong key")
	}
}

func TestTrusteeVerifyPayloadWrongSignature(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
		return
	}
	payload := []byte("this is a random payload")
	// validate signed payload with with same trustee instance
	nodekey = &ecdsa.PrivateKey{}
	// change signature slightly
	signature := trustee.sign(payload)
	if trustee.verify(payload, signature[1:], crypto.FromECDSAPub(&nodekey.PublicKey)) {
		t.Errorf("signature verification did not detect wrong signature")
	}
}

func TestTrusteeVerifyPayloadWrongPayload(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
		return
	}
	payload := []byte("this is a random payload")
	// validate signed payload with with same trustee instance
	nodekey = &ecdsa.PrivateKey{}
	// change payload slightly
	if trustee.verify(payload[1:], trustee.sign(payload), crypto.FromECDSAPub(&nodekey.PublicKey)) {
		t.Errorf("signature verification did not detect wrong payload")
	}
}

func TestTrusteeCredit(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	// build a mock block
	block := newMockBlock("miner", nil)
	// process a credit
	credit(block, []byte("account"), "34343")
	if val, _ := block.Lookup([]byte("account")); val == nil {
		t.Errorf("account key not found")
	} else {
		parsed, _ := strconv.ParseUint(string(val), 10, 64)
		if parsed != 34343 {
			t.Errorf("incorrect amount: %s", val)
		}
	}
}

func TestTrusteeNewMiningRewardTx(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)
	if trustee == nil {
		t.Errorf("failed to get trustee instance")
		return
	}
	// build a mock block
	block := newMockBlock("miner", []string{"uncle1", "uncle2"})
	tx := trustee.NewMiningRewardTx(block)
	if tx == nil {
		t.Errorf("failed to make mining reward transaction")
	}
	// validate signed transaction with with same trustee instance
	if !trustee.verify(tx.Payload, tx.Signature, tx.Submitter) {
		t.Errorf("failed to validate transaction signature using trustee")
	}
	// deserialize transaction
	var ops []Op
	if err := common.Deserialize(tx.Payload, &ops); err != nil {
		t.Errorf("failed to desrialize transaction payload: %s", err)
		return
	}
	// first op in transaction should be self mining award
	for key, value := range ops[0].Params {
		if key == ParamMiner {
			if value != bytesToHexString(trustee.myAddress) {
				t.Errorf("first transaction is not credited to self")
			}
		} else if key == ParamAward {
			if value != MinerAward {
				t.Errorf("incorrect amount for self mining reward: %s", value)
			}
		} else {
			t.Errorf("incorrect parameters: %s -> %s", key, value)
		}
	}
	// second op in transaction should be uncle1's mining reward
	for i, uncle := range[][]byte{[]byte("uncle1"), []byte("uncle2")} {
		for key, value := range ops[i+1].Params {
			if key == ParamUncle {
				if value != bytesToHexString(uncle) {
					t.Errorf("second transaction is not credited to %s", uncle)
				}
			} else if key == ParamAward {
				if value != UncleAward {
					t.Errorf("incorrect amount for uncle mining reward: %s", value)
				}
			} else {
				t.Errorf("incorrect parameters: %s -> %s", key, value)
			}
		}
	}
}

func TestTrusteeVerifyMiningRewardTx(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey1, _ := crypto.GenerateKey()
	nodekey2, _ := crypto.GenerateKey()
	config := &network.AppConfig{}
	trustee1 := NewTrusteeApp(&mgr, nodekey1, config)
	trustee2 := NewTrusteeApp(&mgr, nodekey2, config)

	// build a mock block
	block := newMockBlock("miner", []string{"uncle1", "uncle2"})
	// add new mining transaction from trustee1 to block
	tx := trustee1.NewMiningRewardTx(block)
	block.AddTransaction(tx)
	// set initial values in block
	block.Update([]byte(bytesToHexString(trustee1.myAddress)), []byte("3460000"))
	block.Update([]byte(bytesToHexString([]byte("uncle1"))), []byte("40000"))
	// verify mining transaction with trustee2
	log.SetLogLevel(log.DEBUG)
	if !trustee2.VerifyMiningRewardTx(block) {
		t.Errorf("mining reward transaction validation failed")
	}
	val, _ := block.Lookup([]byte(bytesToHexString(trustee1.myAddress)))
	number, _ := strconv.ParseUint(string(val), 10, 64)
	if number != 4460000 {
		t.Errorf("miner's reward not updated: %s", val)
	}
	val, _ = block.Lookup([]byte(bytesToHexString([]byte("uncle1"))))
	number, _ = strconv.ParseUint(string(val), 10, 64)
	if number != 240000 {
		t.Errorf("uncle1's reward not updated: %s", val)
	}
	val, _ = block.Lookup([]byte(bytesToHexString([]byte("uncle2"))))
	number, _ = strconv.ParseUint(string(val), 10, 64)
	if number != 200000 {
		t.Errorf("uncle2's reward not updated: %s", val)
	}
}
