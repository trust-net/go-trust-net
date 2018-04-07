package consensus

import (
	"time"
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/core"
)

type Transaction struct {
	Payload []byte
	Signature []byte
	Submitter []byte
	Timestamp *core.Byte8
}

func (tx *Transaction) Id() *core.Byte64 {
	hash := sha512.Sum512(tx.Bytes())
	return core.BytesToByte64(hash[:])
}

func (tx *Transaction) Bytes() []byte {
	data := make([]byte, 0, len(tx.Payload)+8+64)
	data = append(data, tx.Payload...)
	data = append(data, tx.Signature...)
	data = append(data, tx.Submitter...)
	data = append(data, tx.Timestamp.Bytes()...)
	return data
}

func NewTransaction(payload, signature, submitter []byte) *Transaction {
	tx := Transaction {
		Payload: make([]byte, 0, len(payload)),
		Signature: make([]byte, 0, len(signature)),
		Submitter: make([]byte, 0, len(submitter)),
		Timestamp: core.Uint64ToByte8(uint64(time.Now().UnixNano())),
	}
	tx.Payload = append(tx.Payload, payload...)
	tx.Signature = append(tx.Signature, signature...)
	tx.Submitter = append(tx.Submitter, submitter...)
	return &tx
}