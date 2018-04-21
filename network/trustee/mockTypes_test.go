package trustee

import (
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/network"
	"github.com/trust-net/go-trust-net/consensus"
)

type MockPlatformManager struct {
	IsStartCalled bool
	StartError error
	IsStopCalled bool
	StopError error
	IsSuspendCalled bool
	SuspendError error
	StatusCallTxId *core.Byte64
	StatusBlock consensus.Block
	StatusError error
	IsStateCalled bool
	StateState *network.State
	SubmitPayload []byte
	SubmitSignature []byte
	SubmitSubmitter []byte
	SubmitId *core.Byte64
	IsPeersCalled bool
	PeersPeers []network.AppConfig
	DisconnectApp *network.AppConfig
	DisconnectError error
}

// start the platform block producer
func (mgr *MockPlatformManager) Start() error {
	mgr.IsStartCalled = true
	return mgr.StartError
}
// stop the platform processing
func (mgr *MockPlatformManager) Stop() error {
	mgr.IsStopCalled = true
	return mgr.StopError
}
// suspend the platform block producer
func (mgr *MockPlatformManager) Suspend() error {
	mgr.IsSuspendCalled = true
	return mgr.SuspendError
}
// query status of a submitted transaction, by its transaction ID
// returns the block where it was finalized, or error if not finalized
func (mgr *MockPlatformManager) Status(txId *core.Byte64) (consensus.Block, error) {
	mgr.StatusCallTxId = txId
	return mgr.StatusBlock, mgr.StatusError
}

// get a snapshot of current world state
func (mgr *MockPlatformManager) State() *network.State {
	mgr.IsStateCalled = true
	return mgr.StateState
}

// submit a transaction payload, and get a transaction ID
func (mgr *MockPlatformManager) Submit(txPayload, signature, submitter []byte) *core.Byte64 {
	mgr.SubmitPayload = txPayload
	mgr.SubmitSignature = signature
	mgr.SubmitSubmitter = submitter
	return mgr.SubmitId
	
}

// get a list of current peers
func (mgr *MockPlatformManager) Peers() []network.AppConfig {
	mgr.IsPeersCalled = true
	return mgr.PeersPeers
}

// disconnect a specific peer
func (mgr *MockPlatformManager) Disconnect(app *network.AppConfig) error {
	mgr.DisconnectApp = app
	return mgr.DisconnectError
}

type mockBlock struct {
	consensus.BlockSpec
	uncleMiners [][]byte
	hash *core.Byte64
	props map[string][]byte
}

func (b *mockBlock) ParentHash() *core.Byte64 {
	return &b.PHASH
}

func (b *mockBlock) Miner() []byte {
	return b.MINER
}

func (b *mockBlock) Nonce() *core.Byte8 {
	return &b.NONCE
}

func (b *mockBlock) Timestamp() *core.Byte8 {
	return &b.TS
}

func (b *mockBlock) Delta() *core.Byte8 {
	return &b.DELTA
}

func (b *mockBlock) Depth() *core.Byte8 {
	return &b.DEPTH
}

func (b *mockBlock) Weight() *core.Byte8 {
	return &b.WT
}

func (b *mockBlock) Update(key, value []byte) bool {
	b.props[string(key)] = value
	return true
}

func (b *mockBlock) Delete(key []byte) bool {
	return false
}

func (b *mockBlock) Lookup(key []byte) ([]byte, error) {
	return b.props[string(key)], nil
}

func (b *mockBlock) Uncles() []core.Byte64 {
	return b.UNCLEs
}

func (b *mockBlock) UncleMiners() [][]byte {
	return b.uncleMiners
}

func (b *mockBlock) Transactions() []consensus.Transaction {
	return b.TXs
}

func (b *mockBlock) AddTransaction(tx *consensus.Transaction) error {
	b.TXs = append(b.TXs, *tx)
	return nil
}

func (b *mockBlock) Hash() *core.Byte64 {
	return b.hash
}

// create a copy of block sendable on wire
func (b *mockBlock) Spec() consensus.BlockSpec {
	spec := consensus.BlockSpec{
		PHASH: b.PHASH,
		MINER: append([]byte{}, b.MINER...),
		TXs: make([]consensus.Transaction,len(b.TXs)),
		TS: b.TS,
		DELTA: b.DELTA,
		DEPTH: b.DEPTH,
		WT: b.WT,
		UNCLEs: make([]core.Byte64,len(b.UNCLEs)),
		NONCE: b.NONCE,
	}
	spec.STATE = b.STATE
	for i, tx := range b.TXs {
		spec.TXs[i] = tx
	}
	for i, uncle := range b.UNCLEs {
		spec.UNCLEs[i] = uncle
	}
	return spec
}

// a deterministic numeric value for the block for ordering of competing blocks 
func (b *mockBlock) Numeric() uint64 {
	num := uint64(0)
	if b.hash == nil {
		num -= 1
		return num
	}
	for _, b := range b.hash.Bytes() {
		num += uint64(b)
	}
	return num
}

func newMockBlock(miner string, uncles []string) *mockBlock {
	block := &mockBlock{
		BlockSpec: consensus.BlockSpec{
			MINER: []byte(miner),
		},
		props: make(map[string][]byte),
	}
	for _, uncle := range uncles {
		block.uncleMiners = append(block.uncleMiners, []byte(uncle))
	}
	return block
}