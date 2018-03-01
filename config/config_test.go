package config

import (
    "testing"
    "fmt"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func TestHappyPathInitialization(t *testing.T) {
	configFile := "testConfig.json"
	config, err := NewConfig(&configFile, nil, nil)
	if err != nil {
		t.Errorf("Failed to create config '%s'", err.Error())
		return
	}
	if config.Bootnodes()[0].String() != "enode://6cc4ce8db4e989e88a4591c727aff984e8c2263284c5e9ca36226c3a6dee15d10d718a6eaa0967e5a98974291f05baf571873e84b8862e0564128f91cc1ed19a@67.169.5.2:32323" {
		t.Errorf("Unexpected value for bootnodes', Found '%s'", config.Bootnodes())
	}
	if *config.NodeName() != "test node" {
		t.Errorf("Unexpected default value for Name', Found '%s'", *config.NodeName())
	}
	if *config.NetworkId() != "0.0.0.0" {
		t.Errorf("Unexpected default value for NetworkId', Found '%s'", *config.NetworkId())
	}
	if *config.DataDir() != "tmp" {
		t.Errorf("Unexpected default value for DataDir', Found '%s'", *config.DataDir())
	}
	expected := "3661376330396163373461343666633036613661326438623532316132613762333062613863613065333963656461356439363061626563306362356533396639643063643139393162326534386637326536313337323630333039353762366366386639613934303431306466643264313463643164353432333761346134"
	hex := fmt.Sprintf("%x", discover.PubkeyID(&config.Key().PublicKey))
	if hex != expected {
		t.Errorf("Unexpected Key: Expected '%s' Found '%s'", expected, hex)
	}
	if config.Port() != nil {
		t.Errorf("Unexpected default value for Port', Found '%s'", config.Port())
	}
	if config.NatEnabled() != nil {
		t.Errorf("Unexpected default value for nat enabled flag', Found '%s'", config.NatEnabled())
	}
}

func TestSetters(t *testing.T) {
	configFile := "testConfig.json"
	port := "1234"
	natEnabled := true
	config, err := NewConfig(&configFile, &port, &natEnabled)
	if err != nil {
		t.Errorf("Failed to create config '%s'", err.Error())
		return
	}
	if *config.Port() != port {
		t.Errorf("Unexpected value for Port', Found '%s'", config.Port())
	}
	if *config.NatEnabled() != true {
		t.Errorf("Unexpected  value for nat enabled flag', Found '%s'", config.NatEnabled())
	}
}

func TestNullConfigFile(t *testing.T) {
	configFile := "nullconfig.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect null config file")
	} else if err.(*ConfigError).Code() != ERR_NULL_CONFIG {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestInvalidConfigData(t *testing.T) {
	configFile := "invalidConfig.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect invalid config data")
	} else if err.(*ConfigError).Code() != ERR_INVALID_CONFIG {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}


func TestInvalidBootnode(t *testing.T) {
	configFile := "invalidBootnode.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect invalid bootnode data")
	} else if err.(*ConfigError).Code() != ERR_INVALID_BOOTNODE {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_BOOTNODE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestMissingKeyFile(t *testing.T) {
	configFile := "missingKeyfile.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect missing key filE")
	} else if err.(*ConfigError).Code() != ERR_MISSING_PARAM {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_MISSING_PARAM, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestInvalidKeyFile(t *testing.T) {
	configFile := "invalidKeyfile.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect missing key filE")
	} else if err.(*ConfigError).Code() != ERR_INVALID_SECRET_FILE {
		fmt.Printf("Got error: %s\n", err.Error())
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_SECRET_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestNewKeyFile(t *testing.T) {
	configFile := "newKeyfile.json"
	config, err := NewConfig(&configFile, nil, nil)
	if err != nil {
		t.Errorf("Failed to create config '%s'", err.Error())
		return
	}
	if config.Key() == nil {
		t.Errorf("key file not generated")
	}
}

func TestMandatoryParamDataDir(t *testing.T) {
	configFile := "noDataDirectory.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect missing data dir")
		return
	} else if err.(*ConfigError).Code() != ERR_MISSING_PARAM {
		fmt.Printf("Got error: %s\n", err.Error())
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_MISSING_PARAM, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestMandatoryParamNodeName(t *testing.T) {
	configFile := "noNodeName.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect missing node name")
		return
	} else if err.(*ConfigError).Code() != ERR_MISSING_PARAM {
		fmt.Printf("Got error: %s\n", err.Error())
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_MISSING_PARAM, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestMandatoryParamNetworkId(t *testing.T) {
	configFile := "noNetworkId.json"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect missing network ID")
		return
	} else if err.(*ConfigError).Code() != ERR_MISSING_PARAM {
		fmt.Printf("Got error: %s\n", err.Error())
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_MISSING_PARAM, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestInvalidConfigFile(t *testing.T) {
	configFile := "invalidfile"
	if _, err := NewConfig(&configFile, nil, nil); err == nil {
		t.Errorf("Did not detect invalid config file")
	} else if err.(*ConfigError).Code() != ERR_INVALID_FILE {
		fmt.Printf("Got error: %s\n", err.Error())
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}