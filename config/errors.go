package config

import (

)

const (
	ERR_INVALID_FILE = 0x01
	ERR_NULL_CONFIG = 0x02
	ERR_INVALID_CONFIG = 0x03
)

type ConfigError struct {
	msg string
	code int
}

func (err *ConfigError) Error() string {
	return err.msg
}

func (err *ConfigError) Code() int {
	return err.code
}

func NewConfigError(code int, msg string) error {
	err := ConfigError {
		msg: msg,
		code: code,
	}
	return &err
}
