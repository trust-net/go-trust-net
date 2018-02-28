package config

import (
	"os"
	"crypto/ecdsa"
	"encoding/json"
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
	if file, err := os.Open(*configFile); err == nil {
		data := make([]byte, 1024)
		if count, err := file.Read(data); err == nil && count <= 1024 {
			data = data[:count]
			params := struct{}{}
			if err := json.Unmarshal(data, &params); err != nil {
				return nil, NewConfigError(ERR_INVALID_CONFIG, err.Error());
			} else {
				return &Config{}, nil
			}
		} else {
			return nil, NewConfigError(ERR_NULL_CONFIG, err.Error());
		}
	} else {
		return nil, NewConfigError(ERR_INVALID_FILE, err.Error());
	}
}