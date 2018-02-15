package core

import (

)

type CoreError struct {
	msg string
}

func (err *CoreError) Error() string {
	return err.msg
}

func NewCoreError(msg string) error {
	err := CoreError {
		msg: msg,
	}
	return &err
}
