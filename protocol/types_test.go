package protocol

import (
    "testing"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/p2p/discover"
    "github.com/satori/go.uuid"
)

func TestBytesToByte32(t *testing.T) {
	expected := []byte{'0','1','2','3','4','5','6','7','8','9','a','b','c','d','e','f','g','h','i','j'}
	actual := BytesToByte32(expected)
	
	// we should get 32 bytes
	if len(actual) != 32 {
		t.Errorf("Expected Hash Length: %d, Actual: %d", HashLength, len(actual))
	}
	i := 0
	for ;i < len(expected); i++ {
		if expected[i] != actual[i] {
			t.Errorf("Expected: %s, Actual: %s", expected, actual)
		}
	}
	for ;i < HashLength; i++ {
		if  actual[i] != 0 {
			t.Errorf("Expected: %d, Actual: %d", expected, actual)
		}
	}
}

func TestBytesToByte16WithExact16Bytes(t *testing.T) {
	expected, _ := uuid.NewV1()
	actual := BytesToByte16(expected.Bytes())
	
	// we should get exact match
	for i, b := range(expected) {
		if b != actual[i] {
			t.Errorf("Expected: %d, Actual: %d", expected.String(), actual)
		}
	}
}

func TestBytesToByte16WithMoreThan16Bytes(t *testing.T) {
	origUuid, _ := uuid.NewV1()
	extra := []byte{'a', 'b', 'c', 'd'}
	expected := append(origUuid.Bytes(), extra...)
	actual := BytesToByte16(expected)

	// we should only get highest 16 bytes
	if len(actual) != 16 {
		t.Errorf("Expected UUID Length: %d, Actual: %d", UUIDLength, len(actual))
	}
	for i := 0; i < UUIDLength; i++ {
		if expected[i] != actual[i] {
			t.Errorf("Expected: %d, Actual: %d", expected, actual)
		}
	}
}

func TestBytesToByte16WithLessThan16Bytes(t *testing.T) {
	short := []byte{'a', 'b', 'c', 'd'}
	extra := make([]byte, 12)
	expected := append(short, extra...)
	actual := BytesToByte16(short)

	// we should get full 16 bytes, padded with zero
	if len(actual) != 16 {
		t.Errorf("Expected UUID Length: %d, Actual: %d", UUIDLength, len(actual))
	}
	for i := 0; i < UUIDLength; i++ {
		if expected[i] != actual[i] {
			t.Errorf("Expected: %d, Actual: %d", expected, actual)
		}
	}
}

func TestBytesToByte64(t *testing.T) {
	key, _ := crypto.GenerateKey()
	expected := discover.PubkeyID(&key.PublicKey).Bytes()
	actual := BytesToByte64(expected)

	// we should get exact 64 bytes
	if len(actual) != 64 {
		t.Errorf("Expected Node ID Length: %d, Actual: %d", EnodeLength, len(actual))
	}
	for i, b := range(expected) {
		if b != actual[i] {
			t.Errorf("Expected: %d, Actual: %d", expected, actual)
		}
	}
}

