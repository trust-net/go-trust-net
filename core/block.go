package core

import (
	"time"
	"fmt"
	"crypto/sha512"
)

// interface to implement transactions
type Transaction interface {
	// execute using input data, and produce and output or error
	Execute(input interface{}) (interface{}, error)
}

// interface to implement block headers
type Header interface {
	Bytes() []byte
}

// interface for node information 
type NodeInfo interface{
	Id()	 string
}

// interface for block specification
type Block interface {
	Previous() Header
	Miner() NodeInfo
	Nonce() Header
	TD() time.Duration
	TXs() []Transaction
	Genesis() Header
	Since() time.Duration
	Hash() []byte
}

// a simple blockchain spec implementation
type SimpleBlock struct {
	previous Header
	miner NodeInfo
	hash []byte
	nonce Header
	genesis Header
	txs	[]Transaction
	td	  time.Duration
	since time.Duration
}

func (b *SimpleBlock) Previous() Header {
	return b.previous
}

func (b *SimpleBlock) Miner() NodeInfo {
	return b.miner
}

func (b *SimpleBlock) Nonce() Header {
	return b.nonce
}

func (b *SimpleBlock) TD() time.Duration {
	return b.td
}

func (b *SimpleBlock) Since() time.Duration {
	return b.since
}

func (b *SimpleBlock) Genesis() Header {
	return b.genesis
}

func (b *SimpleBlock) TXs() []Transaction {
	return b.txs
}

func (b *SimpleBlock) Hash() []byte {
	return b.hash
}

type SimpleHeader struct {
	header []byte
}

func (h *SimpleHeader) Bytes() []byte {
	return h.header
}

func NewSimpleHeader(header string) *SimpleHeader {
	return &SimpleHeader {
		header: []byte(header),
	}
}

func (b *SimpleBlock) ComputeHash() {
	data := make([]byte,0)
	data = append(data, b.previous.Bytes()...)
	data = append(data, []byte(b.miner.Id())...)
	data = append(data, b.genesis.Bytes()...)
	data = append(data, []byte(fmt.Sprintf("%d",b.td))...)
	hash := sha512.Sum512(data)
	b.hash = hash[:]
}

type SimpleNodeInfo struct {
	id string
}

func (n *SimpleNodeInfo) Id() string {
	return n.id
}

func NewSimpleNodeInfo(id string) *SimpleNodeInfo {
	return &SimpleNodeInfo{
		id: id,
	}
}

func NewSimpleBlock(previous Header, genesis Header, since time.Duration, miner NodeInfo) *SimpleBlock {
	return &SimpleBlock{
		previous: previous,
		miner: miner,
		genesis: genesis,
		since: since,
		td: time.Duration(time.Now().UnixNano()) - since,
		txs: make([]Transaction,0),
		hash: make([]byte,0),
	}
}
