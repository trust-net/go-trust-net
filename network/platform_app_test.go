package network

import (
    "testing"
    "time"
    "fmt"
	"github.com/trust-net/go-trust-net/db"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/log"
)

// this test is disabled since cannot get the two apps to connect with each other
func testTwoApps(t *testing.T) {
	log.SetLogLevel(log.DEBUG)
	defer log.SetLogLevel(log.NONE)
	db1, _ := db.NewDatabaseInMem()
	db2, _ := db.NewDatabaseInMem()
	conf1 := testNetworkConfig(nil, nil, nil)
	conf2 := testNetworkConfig(nil, nil, nil)
	var mgr1 PlatformManager
	var mgr2 PlatformManager
	var err error
	if mgr1, err = NewPlatformManager(&conf1.AppConfig, &conf1.ServiceConfig, db1); err != nil {
		t.Errorf("Failed to create platform manager 1: %s", err)
	}
	if err = mgr1.Start(); err != nil {
		t.Errorf("Failed to start platform manager 1: %s", err)
	}
//	// wire 1st app as boot node into 2nd app
//	app := mgr1.(*platformManager)
//	appId := fmt.Sprintf("enode://%x@192.168.1.114:%d", *app.config.minerId, app.srv.NodeInfo().Ports.Discovery)
//	fmt.Printf("App: '%s'\n", appId)
//	conf2.BootstrapNodes = []string{appId}
	if mgr2, err = NewPlatformManager(&conf2.AppConfig, &conf2.ServiceConfig, db2); err != nil {
		t.Errorf("Failed to create platform manager 2: %s", err)
	}
	if err = mgr2.Start(); err != nil {
		t.Errorf("Failed to start platform manager 2: %s", err)
	}
	// connect the two apps
	mgr2.(*platformManager).srv.AddPeer(mgr1.(*platformManager).srv.Self())
	// sleep for some time, for peers to connect
	time.Sleep(1000 * time.Millisecond)
	// submit transaction to mgr1
	txPayload := []byte("test tx payload")
	txSubmitter := []byte("test rx submitter")
	txSignature := []byte("test rx signature")
	txId := mgr1.Submit(txPayload, txSignature, txSubmitter)
	// sleep for some time, for transaction to be processed
	time.Sleep(1000 * time.Millisecond)
	if _, err = mgr2.Status(txId); err != nil {
		t.Errorf("Failed to get transaction updated on mgr 2: %s", err)
	}
	if err = mgr1.Stop(); err != nil {
		t.Errorf("Failed to stop platform manager 1: %s", err)
	}
	if err = mgr2.Stop(); err != nil {
		t.Errorf("Failed to stop platform manager 2: %s", err)
	}
}

func TestPlatformManagerState(t *testing.T) {
	log.SetLogLevel(log.NONE)
	defer log.SetLogLevel(log.NONE)
	db, _ := db.NewDatabaseInMem()
	// define a transaction payload structure
	type testTx struct {
		Op string
		Target string
		Delta int64
	}
	// build a transaction processor using above defined payload
	txProcessor := func(txs *Transaction) bool{
		var opCode testTx
		if err := common.Deserialize(txs.Payload(), &opCode); err != nil {
			fmt.Printf("Failed to serialize payload: %s\n", err)
			return false
		} else {
			currVal := int64(0)
			if val, err := txs.Lookup([]byte(opCode.Target)); err == nil {
				currVal = int64(core.BytesToByte8(val).Uint64())
			}
			switch opCode.Op {
				case "incr":
					currVal += opCode.Delta
				case "decr":
					currVal -= opCode.Delta
			 	default:
				 	fmt.Printf("Unknown opcode: %s\n", opCode.Op)
				 	return false
			}
			fmt.Printf("%s: %d\n", opCode.Target, currVal)
			return txs.Update([]byte(opCode.Target), core.Uint64ToByte8(uint64(currVal)).Bytes())
		}
	}
	conf := testNetworkConfig(txProcessor, nil, nil)
	var mgr PlatformManager
	var err error
	if mgr, err = NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, db); err != nil {
		t.Errorf("Failed to create platform manager: %s", err)
	}
	if err = mgr.Start(); err != nil {
		t.Errorf("Failed to start platform manager: %s", err)
	}
	// submit transactions
	txs := []testTx{
		testTx{"incr", "mangoes", 10},
		testTx{"incr", "bananas", 20},
		testTx{"decr", "mangoes", 2},
		testTx{"incr", "oranges", 5},
		testTx{"decr", "bananas", 5},
	}
	txIds := make([]core.Byte64, 0, len(txs))
	txSubmitter := []byte("test tx submitter")
	for _, tx := range txs {
		txPayload, _ := common.Serialize(tx)
		txSignature := []byte("test rx signature")
		txIds = append(txIds, *mgr.Submit(txPayload, txSignature, txSubmitter))
		// sleep for some time, for transaction to be processed
		time.Sleep(100 * time.Millisecond)
	}
	// sleep some more, for things to stabalize
	time.Sleep(100 * time.Millisecond)
	// verify that transactions have finalized
	for _, txId := range txIds {
		if _, err = mgr.Status(&txId); err != nil {
			t.Errorf("Failed to get submitted transaction status: %s", err)
		}
	}
	// get current state
	state := mgr.State()
	if val, err := state.Get([]byte("mangoes")); err != nil {
		t.Errorf("Failed to get value for mangoes: %s", err)
	} else {
		currVal := int64(core.BytesToByte8(val).Uint64())
		if currVal != 8 {
			t.Errorf("incorrect value for mangoes: %d", currVal)
		}
	}
	if val, err := state.Get([]byte("bananas")); err != nil {
		t.Errorf("Failed to get value for bananas: %s", err)
	} else {
		currVal := int64(core.BytesToByte8(val).Uint64())
		if currVal != 15 {
			t.Errorf("incorrect value for bananas: %d", currVal)
		}
	}
	if val, err := state.Get([]byte("oranges")); err != nil {
		t.Errorf("Failed to get value for oranges: %s", err)
	} else {
		currVal := int64(core.BytesToByte8(val).Uint64())
		if currVal != 5 {
			t.Errorf("incorrect value for oranges: %d", currVal)
		}
	}
	if _, err := state.Get([]byte("peaches")); err == nil {
		t.Errorf("did not expect to get any peaches")
	}
	if _, err := state.GetAllKeys(); err == nil || err.(*core.CoreError).Code() != ERR_NOT_IMPLEMENTED {
		t.Errorf("did not expect get all keys: %s", err)
	}
	if err = mgr.Stop(); err != nil {
		t.Errorf("Failed to stop platform manager: %s", err)
	}
}
