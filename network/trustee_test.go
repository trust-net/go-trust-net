package network

import (
	"fmt"
//	"strconv"
    "testing"
	"crypto/ecdsa"
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNewTrusteeAppInterface(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &AppConfig{}
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
	config := &AppConfig{}
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
//	config := &AppConfig{}
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
	config := &AppConfig{}
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
	config := &AppConfig{}
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
	config := &AppConfig{}
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
	config := &AppConfig{}
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
	if err := Credit(block, []byte("account"), Uint64ToRtu(34343)); err != nil {
		t.Errorf("account credit did not succeed: %s", err)
	}
	if val, _ := block.Lookup([]byte("account")); val == nil {
		t.Errorf("account key not found")
	} else {
		parsed := BytesToRtu(val).Uint64()
//		parsed, _ := strconv.ParseUint(string(val), 10, 64)
		if parsed != 34343 {
			t.Errorf("incorrect amount: %s", val)
		}
	}
}

func TestTrusteeDebit(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	// build a mock block
	block := newMockBlock("miner", nil)
	// add some funds
	if err := Credit(block, []byte("account"), Uint64ToRtu(1000)); err != nil {
		t.Errorf("account credit did not succeed: %s", err)
	}
	// process a debit
	if err := Debit(block, []byte("account"), Uint64ToRtu(100)); err != nil {
		t.Errorf("account debit did not succeed: %s", err)
	}
	if val, _ := block.Lookup([]byte("account")); val == nil {
		t.Errorf("account key not found")
	} else {
		parsed := BytesToRtu(val).Uint64()
//		parsed, _ := strconv.ParseUint(string(val), 10, 64)
		if parsed != 900 {
			t.Errorf("incorrect amount: %s", val)
		}
	}
}

func TestTrusteeDebitInsufficientFunds(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	// build a mock block
	block := newMockBlock("miner", nil)
	// add some funds
	if err := Credit(block, []byte("account"), Uint64ToRtu(1000)); err != nil {
		t.Errorf("account credit did not succeed: %s", err)
	}
	if val, _ := block.Lookup([]byte("account")); val == nil {
		t.Errorf("account key not found")
	} else {
		fmt.Printf("Credited amount: %d\n", BytesToRtu(val).Uint64())
	}
	// process a debit
	if err := Debit(block, []byte("account"), Uint64ToRtu(1001)); err == nil || err.(*core.CoreError).Code() != ERR_LOW_BALANCE {
		t.Errorf("account debit did not detect low balance: %d < %d", Uint64ToRtu(1000).Uint64(), Uint64ToRtu(1001).Uint64())
	}
	// validate that account balance did not change
	if val, _ := block.Lookup([]byte("account")); val == nil {
		t.Errorf("account key not found")
	} else {
		parsed := BytesToRtu(val).Uint64()
//		parsed, _ := strconv.ParseUint(string(val), 10, 64)
		if parsed != 1000 {
			t.Errorf("incorrect amount: %d", parsed)
		}
	}
}

func TestTrusteeNewMiningRewardTx(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &AppConfig{}
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
	config := &AppConfig{}
	trustee1 := NewTrusteeApp(&mgr, nodekey1, config)
	trustee2 := NewTrusteeApp(&mgr, nodekey2, config)

	// build a mock candidate block
	block1 := newMockBlock(string(trustee1.myAddress), []string{"uncle1", "uncle2"})
	// add new mining transaction from trustee1 to candidate block
	tx := trustee1.NewMiningRewardTx(block1)
	// build another mock network block
	block2 := newMockBlock(string(trustee1.myAddress), []string{"uncle1", "uncle2"})
	block2.AddTransaction(tx)
	// set initial values in network block
//	block2.Update([]byte(bytesToHexString(trustee1.myAddress)), []byte("3460000"))
//	block2.Update([]byte(bytesToHexString([]byte("uncle1"))), []byte("40000"))
	block2.Update([]byte(bytesToHexString(trustee1.myAddress)), Uint64ToRtu(3460000).Bytes())
	block2.Update([]byte(bytesToHexString([]byte("uncle1"))), Uint64ToRtu(40000).Bytes())
	// verify mining transaction with trustee2
	if !trustee2.VerifyMiningRewardTx(block2) {
		t.Errorf("mining reward transaction validation failed")
	}
	val, _ := block2.Lookup([]byte(bytesToHexString(trustee1.myAddress)))
	number := BytesToRtu(val).Uint64()
//	number, _ := strconv.ParseUint(string(val), 10, 64)
	if number != 4460000 {
		t.Errorf("miner's reward not updated: %s", val)
	}
	val, _ = block2.Lookup([]byte(bytesToHexString([]byte("uncle1"))))
	number = BytesToRtu(val).Uint64()
//	number, _ = strconv.ParseUint(string(val), 10, 64)
	if number != 240000 {
		t.Errorf("uncle1's reward not updated: %s", val)
	}
	val, _ = block2.Lookup([]byte(bytesToHexString([]byte("uncle2"))))
	number = BytesToRtu(val).Uint64()
//	number, _ = strconv.ParseUint(string(val), 10, 64)
	if number != 200000 {
		t.Errorf("uncle2's reward not updated: %s", val)
	}
}


func TestTrusteeMiningRewardBalance(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	mgr := MockPlatformManager{}
	nodekey, _ := crypto.GenerateKey()
	config := &AppConfig{}
	trustee := NewTrusteeApp(&mgr, nodekey, config)

	// build a mock block
	block := newMockBlock(string(trustee.myAddress), []string{"uncle1", "uncle2"})
	// set mining reward balance value in block
//	block.Update([]byte(bytesToHexString(trustee.myAddress)), []byte("3460000"))
	block.Update([]byte(bytesToHexString(trustee.myAddress)), Uint64ToRtu(3460000).Bytes())
	// ask trustee for the mining award balance
	number := trustee.MiningRewardBalance(block, trustee.myAddress)
	if number != 3460000 {
		t.Errorf("miner's reward not correct: %d", number)
	}
}


func TestUint64ToRtu(t *testing.T) {
	input := uint64(1234567890)
	rtu1 := Uint64ToRtu(input)
	if rtu1.Units != 1234 {
		t.Errorf("incorrect units: %d, expected: %d, from %d / %d", rtu1.Units, input/RtuDivisor, input, RtuDivisor)
	}
	if rtu1.Decimals != 567890 {
		t.Errorf("incorrect decimals: %d, expected: %d, from %d %% %d ", rtu1.Decimals, input%RtuDivisor, input, RtuDivisor)
	}
}

func TestRtuToUint64(t *testing.T) {
	input := uint64(1234567890123456789)
	rtu1 := Uint64ToRtu(input)
	if rtu1.Units != 1234567890123 {
		t.Errorf("incorrect units: %d, expected: %d, from %d / %d", rtu1.Units, input/RtuDivisor, input, RtuDivisor)
	}
	if rtu1.Decimals != 456789 {
		t.Errorf("incorrect decimals: %d, expected: %d, from %d %% %d ", rtu1.Decimals, input%RtuDivisor, input, RtuDivisor)
	}
}

func TestBytes8ToRtu(t *testing.T) {
	input := uint64(2^63 - 1)
	rtu1 := BytesToRtu(core.Uint64ToByte8(input).Bytes())
	if rtu1.Units != input / RtuDivisor {
		t.Errorf("incorrect units: %d", rtu1.Units)
	}
	if rtu1.Decimals != input % RtuDivisor{
		t.Errorf("incorrect decimals: %d", rtu1.Decimals)
	}
}

func TestDoubleBytesToRtu(t *testing.T) {
	units := uint64(1234567890)
	decimals := uint64(987654321)
	rtu1 := BytesToRtu(append(core.Uint64ToByte8(units).Bytes(), core.Uint64ToByte8(decimals).Bytes()...))
	if rtu1.Units != 1234567890987 {
		t.Errorf("incorrect units: %d, expected: %d", rtu1.Units,  1234567890987)
	}
	if rtu1.Decimals != 654321 {
		t.Errorf("incorrect decimals: %d, expected: %d", rtu1.Decimals, 654321)
	}
}

func TestRtuToBytesToRtu(t *testing.T) {
	input := uint64(1234567890987654)
	rtu1 := Uint64ToRtu(input)
	rtu2 := BytesToRtu(rtu1.Bytes())
	if rtu2.Units != 1234567890 {
		t.Errorf("incorrect units: %d, expected: %d, from %d / %d", rtu2.Units, input/RtuDivisor, input, RtuDivisor)
	}
	if rtu2.Decimals != 987654 {
		t.Errorf("incorrect decimals: %d, expected: %d, from %d %% %d ", rtu2.Decimals, input%RtuDivisor, input, RtuDivisor)
	}
}