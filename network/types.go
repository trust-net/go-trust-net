package network

import (
	"crypto/ecdsa"
	"github.com/trust-net/go-trust-net/consensus"
	"github.com/trust-net/go-trust-net/core"
)

type AppConfig struct {
	// identified application's shard/group on public p2p network
	NetworkId	core.Byte16
	// peer's node ID, extracted from p2p connection request
	MinerId		core.Byte64
	// name of the node
	NodeName string
	// application's protocol
	ProtocolId	core.Byte16	
//	// application's authentication/authorization token
//	AuthToken	core.Byte64
}

type TxProcessor	func(txs *Transaction) bool
type PowApprover	func(hash *core.Byte64) bool
type PeerValidator func(config *AppConfig) error

type ServiceConfig struct {
	// application's identity ECDSA key
	IdentityKey *ecdsa.PrivateKey
	// port for application to listen
	Port string
	// NAT traversal preference
	Nat bool
	// application callback to process transactions
	TxProcessor	TxProcessor
	// application callback to approve PoW condition
	PowApprover	PowApprover
	// application callback to validate peer application
	PeerValidator PeerValidator
}
type PlatformConfig struct {
	AppConfig
	ServiceConfig
	// genesis ID for the shard/group
	genesis core.Byte64
}

type Transaction struct {
	payload []byte
	block consensus.Block
}

// provide application submitted transaction payload
func (tx *Transaction) Payload() []byte {
	return tx.payload
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
