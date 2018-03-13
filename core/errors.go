package core

import (

)

const (
	ERR_DUPLICATE_BLOCK = 0x01
	ERR_ORPHAN_BLOCK = 0x02
	ERR_INVALID_HASH = 0x03
	ERR_INVALID_BLOCK = 0x04
	ERR_DB_UNINITIALIZED = 0x05
	ERR_DB_CORRUPTED = 0x06
	ERR_INVALID_UNCLE = 0x07
)

type CoreError struct {
	msg string
	code int
}

func (err *CoreError) Error() string {
	return err.msg
}

func (err *CoreError) Code() int {
	return err.code
}

func NewCoreError(code int, msg string) error {
	err := CoreError {
		msg: msg,
		code: code,
	}
	return &err
}
