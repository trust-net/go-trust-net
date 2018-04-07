package consensus

import (
    "testing"
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/core"
)

func TestTransactionId(t *testing.T) {
	payload := []byte("some data")
	ts := core.Uint64ToByte8(0x12123434)
	id := core.BytesToByte64([]byte("some random submitter"))
	tx := Transaction{
		Payload: payload,
		Timestamp: ts,
		Submitter: id,
	}
	data := make([]byte, 0)
	// first item should be the payload
	data = append(data, payload...)
	// second item should be time stamp
	data = append(data, ts.Bytes()...)
	// third item should be submitter ID
	data = append(data, id.Bytes()...)
	hash := sha512.Sum512(data)
	if hash != *tx.Id() {
		t.Errorf("Hash: Expected: %x, Actual: %x", hash, *tx.Id())
	}
}

func TestTransactionBytes(t *testing.T) {
	payload := []byte("some data")
	ts := core.Uint64ToByte8(0x12123434)
	id := core.BytesToByte64([]byte("some random submitter"))
	tx := Transaction{
		Payload: payload,
		Timestamp: ts,
		Submitter: id,
	}
	data := make([]byte, 0)
	// first item should be the payload
	data = append(data, payload...)
	// second item should be time stamp
	data = append(data, ts.Bytes()...)
	// third item should be submitter ID
	data = append(data, id.Bytes()...)
	bytes := tx.Bytes()
	if len(bytes) != len(data) {
		t.Errorf("Bytes length incorrect: Expected: %d, Actual: %d", len(data), len(bytes))
	}
	for i, b := range bytes {
		if b != data[i] {
			t.Errorf("Bytes incorrect:\nExpected: %x\nActual: %x", data, bytes)
			break
		}
	}
}


func TestNewTransaction(t *testing.T) {
	payload := []byte("some data")
	submitter := core.BytesToByte64([]byte("some random submitter"))
	signature := core.BytesToByte64([]byte("some random signature"))
	tx := NewTransaction(payload, submitter, signature)
	if string(tx.Payload) != "some data" {
		t.Errorf("incorrect payload: '%s'", tx.Payload)
	}
	if *tx.Submitter != *submitter {
		t.Errorf("incorrect submitter: %s", *tx.Submitter)
	}
	if *tx.Signature != *signature {
		t.Errorf("incorrect signature: %s", *tx.Signature)
	}
}
