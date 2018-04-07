package network

import (
	"crypto/ecdsa"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/trust-net/go-trust-net/core"
)

type AppConfig struct {
	// identified application's shard/group on public p2p network
	NetworkId core.Byte16
	// peer's node ID, extracted from p2p connection request
	NodeId string
	// name of the node
	NodeName string
	// application's protocol ID
	ProtocolId	core.Byte16	
//	// application's authentication/authorization token
//	AuthToken	core.Byte64
}

// a callback provided by application to process submitted transactions for inclusion into a new block
type TxProcessor	func(txs *Transaction) bool
// a callback provided by application to approve PoW
// (arguments include block's timestamp and delta time since parent block,
// so that application can implement variable PoW schemes based on time when
// block was generated and time since its parent)
type PowApprover	func(hash []byte, blockTs, parentTs uint64) bool
// a callback provided by application to validate a peer application connection
type PeerValidator func(config *AppConfig) error

type ServiceConfig struct {
	// application's identity ECDSA key
	// (required)
	IdentityKey *ecdsa.PrivateKey
	// port for application to listen
	Port string
	// NAT traversal preference
	Nat bool
	// appication's protocol name
	ProtocolName string
	// appication's protocol version
	ProtocolVersion uint
	// list of bootstrap nodes
	BootstrapNodes []string
	// application callback to process transactions
	// (required)
	TxProcessor	TxProcessor
	// application callback to approve PoW condition
	// (optional: set to nil, if no PoW required)
	PowApprover	PowApprover
	// application callback to validate peer application
	// (required)
	PeerValidator PeerValidator
}
type PlatformConfig struct {
	AppConfig
	ServiceConfig
	// genesis ID for the shard/group
	genesis *core.Byte64
	// miner ID for mining reward, extracted from p2p server
	minerId *core.Byte64
}

// container to access world state information
type State struct {
	block consensus.Block
}

func (s *State) Get(key []byte) ([]byte, error) {
	return s.block.Lookup(key)
}

func (s *State) GetAllKeys() ([][]byte, error) {
	// TODO
	return nil, core.NewCoreError(ERR_NOT_IMPLEMENTED, "TBD")
}

type Transaction struct {
//	payload []byte
	consensus.Transaction
	block consensus.Block
}

// provide application submitted transaction payload
func (tx *Transaction) Payload() []byte {
	return tx.Transaction.Payload
}

// provide submitter's public key
func (tx *Transaction) Submitter() *core.Byte64 {
	return tx.Transaction.Submitter
}

// provide signature of the payload
func (tx *Transaction) Signature() *core.Byte64 {
	return tx.Transaction.Signature
}

// lookup a key in current world state view
func (tx *Transaction) Lookup(key []byte) ([]byte, error) {
	return tx.block.Lookup(key)
}

// delete a key in current world state view
func (tx *Transaction) Delete(key []byte) bool {
	return tx.block.Delete(key)
}

// update a key in current world state view
func (tx *Transaction) Update(key, value []byte) bool {
	return tx.block.Update(key, value)
}
