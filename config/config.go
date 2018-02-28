package config

import (
	"os"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/p2p/discover"

)

// container class for different configuration parameters
type Config struct{
	bootnodes []*discover.Node
	nodeName *string
	networkId *string
	dataDir *string
	key *ecdsa.PrivateKey
	port *string
	natEnabled *bool
}

func (c *Config) Bootnodes() []*discover.Node {
	return c.bootnodes
}

func (c *Config) Name() *string {
	return c.nodeName
}

func (c *Config) NetworkId() *string {
	return c.networkId
}

func (c *Config) DataDir() *string {
	return c.dataDir
}

func (c *Config) Key() *ecdsa.PrivateKey {
	return c.key
}

func (c *Config) Port() *string {
	return c.port
}

func (c *Config) isNatEnabled() *bool {
	return c.natEnabled
}

func (c *Config) SetPort(port *string) {
	c.port = port
}

func (c *Config) SetNatEnabled(natEnabled *bool) {
	c.natEnabled = natEnabled
}

func NewConfig(configFile *string) (*Config, error) {
	if _, err := os.Open(*configFile); err == nil {
		return &Config{}, nil	
	} else {
		return nil, NewConfigError(ERR_INVALID_FILE, err.Error());
	}
}