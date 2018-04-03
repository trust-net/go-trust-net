package network

import (
	"github.com/trust-net/go-trust-net/core"
	"github.com/ethereum/go-ethereum/p2p"
)


var (
//	testGenesis = *core.BytesToByte64([]byte("some test genesis hash"))
	testNetwork = *core.BytesToByte16([]byte("some test network"))
)

type testReader struct {
	response []byte
	err error
}

func (r *testReader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	n = len(p)
	if n < len(r.response) {
		n = len(r.response)
	}
	copy(p, r.response[:n])
	return n, nil
} 

type testMocks struct {
	sendErr error
	sendSucc bool
	readErr error
	readResp p2p.Msg
}

type mockNode struct {
	peerNode
	// test method mocks
	testMocks
}

func (node *mockNode) Send(msgcode uint64, data interface{}) error {
	// mock behavior for testing
	if node.sendErr != nil || node.sendSucc {
		return node.sendErr
	}
	return node.peerNode.Send(msgcode, data)
}

func (node *mockNode) ReadMsg() (p2p.Msg, error) {
	// mock behavior for testing
	if node.readErr != nil || node.readResp.Payload != nil{
		return node.readResp, node.readErr
	}
	return node.peerNode.conn.ReadMsg()
}

func testNetworkConfig(processor TxProcessor, approver PowApprover, validator PeerValidator) *PlatformConfig {
	conf := PlatformConfig {
		AppConfig: AppConfig {
			NetworkId: testNetwork,
		},
		ServiceConfig: ServiceConfig{
			TxProcessor: processor,
			PowApprover: approver,
			PeerValidator: validator,
		},
	}
	return &conf
}

func testPeerHandshakeMsg(myConf *PlatformConfig) *HandshakeMsg {
	msg := &HandshakeMsg{
		NetworkId: myConf.NetworkId,
		Genesis: myConf.genesis,
		ProtocolId: myConf.ProtocolId,		
	}
	return msg
}
