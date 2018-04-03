package network

import (
    "testing"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/log"
	"github.com/ethereum/go-ethereum/p2p"
)

func TestNewPlatformManagerNullArgs(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	if _, err := NewPlatformManager(nil, &ServiceConfig{}, db); err == nil {
		t.Errorf("did not detect nil app config")
	}
	if _, err := NewPlatformManager(&AppConfig{}, nil, db); err == nil {
		t.Errorf("did not detect nil service config")
	}
	if _, err := NewPlatformManager(&AppConfig{}, &ServiceConfig{}, nil); err == nil {
		t.Errorf("did not detect nil app DB")
	}	
}

func TestNewPlatformManagerGoodArgs(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers from new platform instance")
		}
	}
}

func TestValidateAndAddIncorrectNetwork(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		handshake.NetworkId = *core.BytesToByte16([]byte("an invalid network id"))
		peer := &mockNode{}
		if err := mgr.validateAndAdd(handshake, peer); err == nil {
			t.Errorf("Failed to detect network mistmatch in peer handshake")
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers for incorrect handshake")
		}
	}
}

func TestValidateAndAddIncorrectGenesis(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	conf := testNetworkConfig(nil, nil, nil)
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		handshake.Genesis = *core.BytesToByte64([]byte("an invalid genesis"))
		peer := &mockNode{}
		if err := mgr.validateAndAdd(handshake, peer); err == nil {
			t.Errorf("Failed to detect genesis mistmatch in peer handshake")
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers for incorrect handshake")
		}
	}
}

func TestValidateAndAddValidatorError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that will reject peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return core.NewCoreError(0x11, "test error")
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
		}
		if err := mgr.validateAndAdd(handshake, peer); err == nil || err.(*core.CoreError).Code() != 0x11 || err.Error() != "test error" {
			t.Errorf("Failed to detect application validation error in peer handshake: %s", err)
		}
		if mgr.PeerCount() != 0 {
			t.Errorf("did not expect any peers for incorrect handshake")
		}
	}
}

func TestValidateAndAddPeerSetUpdate(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
		}
		if err := mgr.validateAndAdd(handshake, peer); err != nil {
			t.Errorf("Failed to validate peer handshake: %s", err)
		}
		if mgr.PeerCount() != 1 {
			t.Errorf("did not update peer count")
		}
		if mgr.peerDb.PeerNodeForId(peer.Id()) != peer {
			t.Errorf("did not find peer in peer set DB")
		}
	}
}

func TestUnregisterPeer(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
		}
		if err := mgr.validateAndAdd(handshake, peer); err != nil {
			t.Errorf("Failed to validate peer handshake: %s", err)
		}
		// now disconnect peer
		mgr.UnregisterPeer(peer)
		if mgr.PeerCount() != 0 {
			t.Errorf("did not decrement peer count")
		}
		if mgr.peerDb.PeerNodeForId(peer.Id()) != nil {
			t.Errorf("remove peer from peer set DB")
		}
	}
}

func TestHandshakeMsgSendErr(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				sendErr: core.NewCoreError(0x2100000, "test send error"),
			},
		}
		if err := mgr.Handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != ErrorHandshakeFailed {
			t.Errorf("Failed to detect handshake msg send error: %s", err)
			return
		}
	}
}

func TestHandshakeMsgReadErr(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				readErr: core.NewCoreError(0x2200000, "test read error"),
				sendSucc: true,
			},
		}
		if err := mgr.Handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != 0x2200000 {
			t.Errorf("Failed to detect handshake msg read error: %s", err)
			return
		}
	}
}

func TestHandshakeWrongMsg(t *testing.T) {
	log.SetLogLevel(log.DEBUG)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				readResp: p2p.Msg{
					Payload: &testReader{
						response: []byte("some test response"),
					},
					Code: 0x22222,
				},
				sendSucc: true,
			},
		}
		if err := mgr.Handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != ErrorHandshakeFailed {
			t.Errorf("Failed to detect incorrect message code: %s", err)
			return
		}
	}
}

func TestHandshakeDecodeError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// create a network config with validator function that accept peer connection
	conf := testNetworkConfig(nil, nil, func(config *AppConfig) error {
			return nil
	})
	if mgr, err := NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	} else {
		// create a peer handshake message with network ID mismatch
		handshake := testPeerHandshakeMsg(mgr.config)
		peer := &mockNode{
			peerNode: peerNode{
				peer: p2p.NewPeer([64]byte(*core.BytesToByte64([]byte("a random peer node ID"))), "", nil),
			},
			testMocks: testMocks{
				readResp: p2p.Msg{
					Payload: &testReader{
						response: nil,
						err: core.NewCoreError(0x232323, "test decode error"),
					},
					Code: Handshake,
				},
				sendSucc: true,
			},
		}
		if err := mgr.Handshake(handshake, peer); err == nil || err.(*core.CoreError).Code() != ErrorHandshakeFailed {
			t.Errorf("Failed to detect message decode error: %s", err)
			return
		}
	}
}
