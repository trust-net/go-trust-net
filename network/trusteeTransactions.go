package network

import (
	"github.com/trust-net/go-trust-net/core"
)

type Op struct {
	OpCode string `json:"op_code"       gencodec:"required"`
	Params map[string]string `json:"params"       gencodec:"required"`
}

const (
	OpReward = "REWARD"
	ParamMiner = "MINER"
	ParamUncle = "UNCLE"
	ParamAward = "AWARD"
	MinerAward = "1000000"
	UncleAward = "200000"
//	RtuDecimal = uint64(4)
	RtuDivisor = 1000000
)

func NewOp(opCode string) *Op {
	switch opCode {
		case OpReward:
			return &Op{
				OpCode: opCode,
				Params: make(map[string]string),
			}
		default:
			return nil
	}
}

type RTU struct {
	Units uint64
	Decimals uint64
}

func (rtu *RTU) Bytes() []byte {
	return append(core.Uint64ToByte8(rtu.Units).Bytes(), core.Uint64ToByte8(rtu.Decimals).Bytes()...)
}

func (rtu *RTU) Uint64() uint64 {
	return rtu.Units * RtuDivisor + (rtu.Decimals % RtuDivisor)
}

func Uint64ToRtu(number uint64) *RTU {
	rtu := RTU{
		Units: number/RtuDivisor,
		Decimals: number%RtuDivisor,
	}
	return &rtu
}
func BytesToRtu(bytes []byte) *RTU {
	if len(bytes) < 9 {
		return Uint64ToRtu(core.BytesToByte8(bytes).Uint64())
	} else {
		units := core.BytesToByte8(bytes[:8]).Uint64()
		decimals := core.BytesToByte8(bytes[8:]).Uint64()
		divisor := uint64(1)
		for divisor < decimals {
			divisor = divisor * 10
		}
		for divisor > RtuDivisor {
			divisor = divisor / 10
			units = (units * 10) + (decimals / divisor)
			decimals = decimals % divisor
		}
		rtu := RTU{
			Units: units,
			Decimals: decimals,
		}
		return &rtu
	}
}