package protocol

import (
	"fmt"
)

type ProtocolError struct {
	msg string
	code int
}

func (err *ProtocolError) Error() string {
	return fmt.Sprintf("Error [%03d] : %s", err.code, err.msg)
}

func NewProtocolError(code int, msg string) error {
	err := ProtocolError {
		msg: msg,
		code: code,
	}
	return &err
}
