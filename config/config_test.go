package config

import (
    "testing"
    "fmt"
)

func TestHappyPathInitialization(t *testing.T) {
	configFile := "testConfig.json"
	config, err := NewConfig(&configFile)
	if err != nil {
		t.Errorf("Failed to create config '%s'", err.Error())
		return
	}
	if config.Bootnodes() != nil {
		t.Errorf("Unexpected default value for bootnodes', Found '%s'", config.Bootnodes())
	}
	if config.Name() != nil {
		t.Errorf("Unexpected default value for Name', Found '%s'", config.Name())
	}
	if config.NetworkId() != nil {
		t.Errorf("Unexpected default value for NetworkId', Found '%s'", config.NetworkId())
	}
	if config.DataDir() != nil {
		t.Errorf("Unexpected default value for DataDir', Found '%s'", config.DataDir())
	}
	if config.Key() != nil {
		t.Errorf("Unexpected default value for Key', Found '%s'", config.Key())
	}
	if config.Port() != nil {
		t.Errorf("Unexpected default value for Port', Found '%s'", config.Port())
	}
	if config.isNatEnabled() != nil {
		t.Errorf("Unexpected default value for nat enabled flag', Found '%s'", config.isNatEnabled())
	}
}

func TestSetters(t *testing.T) {
	configFile := "testConfig.json"
	config, err := NewConfig(&configFile)
	if err != nil {
		t.Errorf("Failed to create config '%s'", err.Error())
		return
	}
	port := "1234"
	config.SetPort(&port)
	natEnabled := true
	config.SetNatEnabled(&natEnabled)
	if *config.Port() != port {
		t.Errorf("Unexpected value for Port', Found '%s'", config.Port())
	}
	if *config.isNatEnabled() != true {
		t.Errorf("Unexpected  value for nat enabled flag', Found '%s'", config.isNatEnabled())
	}
}

func TestNullConfigFile(t *testing.T) {
	configFile := "nullconfig.json"
	if _, err := NewConfig(&configFile); err == nil {
		t.Errorf("Did not detect null config file")
	} else if err.(*ConfigError).Code() != ERR_NULL_CONFIG {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestInvalidConfigData(t *testing.T) {
	configFile := "invalidConfig.json"
	if _, err := NewConfig(&configFile); err == nil {
		t.Errorf("Did not detect invalid config data")
	} else if err.(*ConfigError).Code() != ERR_INVALID_CONFIG {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}

func TestInvalidConfigFile(t *testing.T) {
	configFile := "invalidfile"
	if _, err := NewConfig(&configFile); err == nil {
		t.Errorf("Did not detect invalid config file")
	} else if err.(*ConfigError).Code() != ERR_INVALID_FILE {
		t.Errorf("Did not detect correct error code: Expected %d, Found %d", ERR_INVALID_FILE, err.(*ConfigError).Code())
	} else {
		fmt.Printf("Got correct error: %s\n", err.Error())
	}
}