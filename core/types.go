package core

import (
	"encoding/binary"
)

const (
	HashLength	= 32
	UUIDLength	= 16
	EnodeLength	= 64
)

type ByteArray interface {
	Bytes() []byte
}

type Byte32 	[HashLength]byte
type Byte16 	[UUIDLength]byte
type Byte64	[EnodeLength]byte
type Byte8 	[8]byte

func (bytes *Byte8) Bytes() []byte {
	return bytes[:]
}

func (bytes *Byte8) Uint64() uint64 {
	return binary.BigEndian.Uint64(bytes[:])
}

func (bytes *Byte32) Bytes() []byte {
	return bytes[:]
}

func (bytes *Byte16) Bytes() []byte {
	return bytes[:]
}

func (bytes *Byte64) Bytes() []byte {
	return bytes[:]
}

func copyBytes(source []byte, dest []byte, size int) {
	i := len(source)
	if i > size {
		i = size
	}
	for ;i > 0; i-- {
		dest[i-1] = source[i-1]
	}
}

func BytesToByte8(source []byte) *Byte8 {
	var byte8 Byte8
	copyBytes(source, byte8.Bytes(), 8)
	return &byte8
}

func Uint64ToByte8(source uint64) *Byte8 {
	var byte8 Byte8
	binary.BigEndian.PutUint64(byte8[:], source)
	return &byte8
}

func BytesToByte16(source []byte) *Byte16 {
	var byte16 Byte16
	copyBytes(source, byte16.Bytes(), UUIDLength)
	return &byte16
}

func BytesToByte32(source []byte) *Byte32 {
	var byte32 Byte32
	copyBytes(source, byte32.Bytes(), HashLength)
	return &byte32
}

func BytesToByte64(source []byte) *Byte64 {
	var byte64 Byte64
	copyBytes(source, byte64.Bytes(), EnodeLength)
	return &byte64
}
