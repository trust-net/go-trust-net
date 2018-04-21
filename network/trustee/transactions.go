package trustee

import (
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
