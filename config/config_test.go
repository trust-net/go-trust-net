package config

import (
    "testing"
)

func TestHappyPathInitialization(t *testing.T) {
	configFile := "some file"
	config := NewConfig(&configFile)
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
		t.Errorf("Unexpected default value for Port', Found '%s'", config.isNatEnabled())
	}
}

