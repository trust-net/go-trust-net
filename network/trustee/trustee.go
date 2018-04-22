package trustee

import (
	"fmt"
	"strconv"
    "math/big"
    "crypto/rand"
	"crypto/ecdsa"
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/network"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/ethereum/go-ethereum/crypto"
)

// a trustee app for trust node mining award management
// we are abstracting the mining award management into an "app" instead of
// directly implementing it as part of network protocol layer, so that
// app can be extended to add token management APIs (e.g. balance query, balance transfer, etc)

type Trustee interface {
	NewMiningRewardTx(block consensus.Block) *consensus.Transaction
	VerifyMiningRewardTx(block consensus.Block) bool
	MiningRewardBalance(block consensus.Block, miner []byte) uint64
}

type trusteeImpl struct {
	log log.Logger
	myMgr network.PlatformManager
	identity *ecdsa.PrivateKey
	appConfig *network.AppConfig
	myAddress []byte
	myReward *Op
}

func bytesToHexString(bytes []byte) string {
	return fmt.Sprintf("%x", bytes)
}

func NewTrusteeApp(mgr network.PlatformManager, identity *ecdsa.PrivateKey, appConfig *network.AppConfig) *trusteeImpl {
	trustee := &trusteeImpl{
		myMgr: mgr,
		identity: identity,
		appConfig: appConfig,
		myAddress: crypto.FromECDSAPub(&identity.PublicKey),
		myReward: NewOp(OpReward),
	}
	trustee.myReward.Params[ParamMiner] = bytesToHexString(trustee.myAddress)
	trustee.myReward.Params[ParamAward] = MinerAward
	trustee.log = log.NewLogger(*trustee)
	trustee.log.Debug("Initialized trustee app for Node: %s@%x|ProtocolId: %x|NetworkId: %x", appConfig.NodeName, trustee.myAddress, appConfig.ProtocolId, appConfig.NetworkId)
	return trustee
}

type signature struct {
	R *big.Int
	S *big.Int
}

func (t *trusteeImpl) sign(payload []byte) []byte {
	// sign the payload
	s := signature{}
	var err error
	// we want to sign the hash of the payload
	hash := sha512.Sum512(payload)
	if s.R,s.S, err = ecdsa.Sign(rand.Reader, t.identity, hash[:]); err != nil {
		t.log.Error("Failed to sign payload: %s", err)
		return nil
	}
	var txSignature []byte
	if txSignature,err = common.Serialize(s); err != nil {
		t.log.Error("Failed to serialize signature: %s", err)
		return nil
	}
	return txSignature
}


func (t *trusteeImpl) verify(payload, sign, submitter []byte) bool {
	// extract submitter's key
	key := crypto.ToECDSAPub(submitter)
	if key == nil {
		return false
	}
	// regenerate signature parameters
	s := signature{}
	if err := common.Deserialize(sign, &s); err != nil {
		t.log.Error("Failed to parse signature: %s", err)
		return false
	}
	// we want to validate the hash of the payload
	hash := sha512.Sum512(payload)
	// validate signature of payload
	return ecdsa.Verify(key, hash[:], s.R, s.S)
}

// generate reward transaction for miner and uncles of the block (BUT DO NOT ADD TRANSACTION TO BLOCK YET)
// (note, uncles here are the node-ID of uncle blocks, prepopulated by the consensus layer when candidate block was created)
func (t *trusteeImpl) NewMiningRewardTx(block consensus.Block) *consensus.Transaction {
	// build list of miner nodes for uncle blocks
	uncleMiners := make([][]byte, len(block.UncleMiners()))
	for i, uncleMiner := range block.UncleMiners() {
		uncleMiners[i] = uncleMiner
	}
	
	ops := make([]Op, 1 + len(uncleMiners))
	// first add self's mining reward
	ops[0] = *t.myReward
	
	// now add award for each uncle
	for i, uncleMiner := range uncleMiners {
		op := NewOp(OpReward)
		op.Params[ParamUncle] = bytesToHexString(uncleMiner)
		op.Params[ParamAward] = UncleAward
		ops[i+1] = *op 
	}
	// serialize ops into payload
	if payload,err := common.Serialize(ops); err != nil {
		t.log.Error("Failed to serialize ops into payload: %s", err)
		return nil
	} else {
		// make a signed transaction out of payload
		if signature := t.sign(payload); len(signature) > 0 {
			// return the signed transaction
			return consensus.NewTransaction(payload, signature, t.myAddress)
		}
	}
	return nil
}

// verify reward transaction for network block's miner and uncles
// (note, uncles here are the node-ID of uncle blocks, pre-populated by the consensus layer when network block was processed)
func (t *trusteeImpl) VerifyMiningRewardTx(block consensus.Block) bool {
	// pop first transaction from the block
	tx := block.Transactions()[0]

	// validate transaction signature
	if !t.verify(tx.Payload, tx.Signature, tx.Submitter) {
		return false
	}

	// validate that mining award is for block's miner
	if string(tx.Submitter) != string(block.Miner()) {
		t.log.Error("mining award owner is not the block miner")
		return false
	}

	// process transaction and update block
	return t.process(block, &tx)
}

// get node's reward balance based on world view of the block
func (t *trusteeImpl) MiningRewardBalance(block consensus.Block, miner []byte) uint64 {
	strVal := "0"
	if val, err := block.Lookup([]byte(bytesToHexString(miner))); err == nil {
		strVal = string(val)
	}
	var value uint64
	var err error
	if value, err = strconv.ParseUint(strVal, 10, 64); err != nil {
		value = 0
	}
	return value
}

func credit(block consensus.Block, account []byte, amount string) {
	strVal := "0"
	if val, err := block.Lookup(account); err == nil {
		strVal = string(val)
	}
	var value, delta uint64
	var err error
	if value, err = strconv.ParseUint(strVal, 10, 64); err != nil {
		value = 0
	}
	if delta, err = strconv.ParseUint(amount, 10, 64); err != nil {
		delta = 0
	}
	value += delta
	strVal = strconv.FormatUint(value, 10)
	block.Update(account, []byte(strVal))
}

func (t *trusteeImpl) process(block consensus.Block, tx *consensus.Transaction) bool {
	// deserialize ops from transaction payload
	var ops []Op
	if err := common.Deserialize(tx.Payload, &ops); err != nil {
		t.log.Error("Failed to de-serialize ops from payload: %s", err)
		return false
	}
	
	// make sure correct number of ops
	if len(ops) != 1 + len(block.UncleMiners()) {
		t.log.Error("incorrect number of ops in mining transaction")
		return false
	}

	// first op in transaction should be self mining award
	validOp := true
	var account []byte
	var amount string
	for key, value := range ops[0].Params {
		if key == ParamMiner {
			if value != bytesToHexString(tx.Submitter) {
				t.log.Error("first transaction is not credited to miner")
				validOp = false
			} else {
				account = []byte(value)
			}
		} else if key == ParamAward {
			if value != MinerAward {
				t.log.Error("incorrect amount for miner reward: %s", value)
				validOp = false
			} else {
				amount = value
			}
		} else {
			t.log.Error("incorrect parameters: %s -> %s", key, value)
			validOp = false
		}
	}
	// if op is valid, credit reward
	if validOp && len(account) > 0 && len(amount) > 0 {
		t.log.Debug("crediting miner %s with %s", account, amount)
		credit(block, account, amount)
	} else {
		return false
	}
	// subsequent ops in transaction should be uncle mining reward
	for i, uncle := range block.UncleMiners() {
		validOp, account, amount = true, nil, "0"
		for key, value := range ops[i+1].Params {
			if key == ParamUncle {
				if value != bytesToHexString(uncle) {
					t.log.Error("uncle reward is not credited to %s", uncle)
					validOp = false
				} else {
					account = []byte(value)
				}
			} else if key == ParamAward {
				if value != UncleAward {
					t.log.Error("incorrect amount for uncle mining reward: %s", value)
					validOp = false
				} else {
					amount = value
				}
			} else {
				t.log.Error("incorrect parameters: %s -> %s", key, value)
				validOp = false
			}
		}
		// if op is valid, credit reward
		if validOp && len(account) > 0 && len(amount) > 0 {
			t.log.Debug("crediting uncle %s with %s", account, amount)
			credit(block, account, amount)
		} else {
			return false
		}
	}
	return true
}
