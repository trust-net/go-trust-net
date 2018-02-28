package config

import (

)

const (
	ERR_INVALID_FILE = 0x01
	ERR_NULL_CONFIG = 0x02
	ERR_INVALID_CONFIG = 0x03
	ERR_INVALID_BOOTNODE = 0x04
	ERR_INVALID_SECRET_FILE = 0x05
	ERR_MARSHAL_FAILURE = 0x06
	ERR_INVALID_SECRET_DATA = 0x07
	ERR_MISSING_PARAM = 0x08
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
