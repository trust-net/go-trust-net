package consensus

import (
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/core"
)

type Transaction struct {
	Payload []byte
	Timestamp *core.Byte8
	Submitter *core.Byte64
}

func (tx *Transaction) Id() *core.Byte64 {
	hash := sha512.Sum512(tx.Bytes())
	return core.BytesToByte64(hash[:])
}


func (tx *Transaction) Bytes() []byte {
	data := make([]byte, 0, len(tx.Payload)+8+64)
	data = append(data, tx.Payload...)
	data = append(data, tx.Timestamp.Bytes()...)
	data = append(data, tx.Submitter.Bytes()...)
	return data
}