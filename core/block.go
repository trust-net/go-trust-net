package core

import (
	"math/big"
)

// interface to implement transactions
type Transaction interface {
	// execute using input data, and produce and output or error
	Execute(input interface{}) interface{}, error
}

// interface to implement block headers
type Header interface {
	String() string
}

// interface for node information 
type NodeInfo interface{
	Id()	 string
}

// interface for block specification
type Block interface {
	Previous() Header
	Miner() NodeInfo
	Hash() Header
	TD() *big.Int
	TXs() []Transaction
}

// a simple blockchain spec implementation
type SimpleBlock struct {
	previous Header
	miner Node
}