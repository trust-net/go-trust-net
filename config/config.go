package config

import (
	"os"
	"sync"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/crypto"

)

// container class for different service configurations
type config struct{
	bootnodes []*discover.Node
	nodeName *string
	networkId *string
	dataDir *string
	key *ecdsa.PrivateKey
	port *string
	natEnabled *bool
}

type configParams struct {
	Bootnodes []*string	`json:"boot_nodes"       gencodec:"required"`
	NodeName *string		`json:"node_name"       gencodec:"required"`
	NetworkId *string	`json:"network_id"       gencodec:"required"`
	DataDir *string		`json:"data_dir"       gencodec:"required"`
	KeyFile *string		`json:"key_file"       gencodec:"required"`
}

type keyPair struct {
	Curve string
	X, Y []byte
	D []byte
}

var c *config
var lock   sync.RWMutex

func Config() (*config, error) {
	if c == nil {
		return nil, NewConfigError(ERR_NOT_INITIALIZED, "service config not initialized")
	}
	return c, nil
} 

func (c *config) Bootnodes() []*discover.Node {
	return c.bootnodes
}

func (c *config) NodeName() *string {
	return c.nodeName
}

func (c *config) NetworkId() *string {
	return c.networkId
}

func (c *config) DataDir() *string {
	return c.dataDir
}

func (c *config) Key() *ecdsa.PrivateKey {
	return c.key
}

func (c *config) Port() *string {
	return c.port
}

func (c *config) NatEnabled() *bool {
	return c.natEnabled
}

func InitializeConfig(configFile *string, port *string, natEnabled *bool) error {
	// check if already initialized
	if c != nil {
		return nil
	}
	lock.Lock()
	defer lock.Unlock()
	if c != nil {
		return nil
	}
	// open the config file
	if file, err := os.Open(*configFile); err == nil {
		data := make([]byte, 1024)
		// read config data from file
		if count, err := file.Read(data); err == nil && count <= 1024 {
			data = data[:count]
			params := configParams{}
			// parse json data into structure
			if err := json.Unmarshal(data, &params); err != nil {
				return NewConfigError(ERR_INVALID_CONFIG, err.Error());
			} else {
				// validate mandatory simple config params
				if params.DataDir == nil {
					return NewConfigError(ERR_MISSING_PARAM, "data directory not specified")
				}
				// TODO, add check for valid and accessible directory
				
				if params.NetworkId == nil {
					return NewConfigError(ERR_MISSING_PARAM, "network ID not specified")
				}
				if params.NodeName == nil {
					return NewConfigError(ERR_MISSING_PARAM, "node name not specified")
				}
				// populate simple config parameters
				config := config {
					nodeName: params.NodeName,
					port: port,
					natEnabled: natEnabled,
					networkId: params.NetworkId,
					dataDir: params.DataDir,
				}
				// parse bootnodes and add to config, if present
				if params.Bootnodes != nil {
					config.bootnodes = make([]*discover.Node,0,len(params.Bootnodes))
					for _, bootnode := range params.Bootnodes {
						if enode, err := discover.ParseNode(*bootnode); err == nil {
							config.bootnodes = append(config.bootnodes, enode)
						} else {
							return NewConfigError(ERR_INVALID_BOOTNODE, err.Error())
						}
					}
				}
				// parse secret key file, if present, else generate new secret key and persist
				if params.KeyFile == nil {
					return NewConfigError(ERR_MISSING_PARAM, "key filename not specified")
				}
				if file, err := os.Open(*params.KeyFile); err == nil {
					// source the secret key from file
					data := make([]byte, 1024)
					if count, err := file.Read(data); err == nil && count <= 1024 {
						data = data[:count]
						kp := keyPair{}
						if err := json.Unmarshal(data, &kp); err != nil {
							return NewConfigError(ERR_INVALID_SECRET_DATA, err.Error())
						} else {
							nodekey := new(ecdsa.PrivateKey)
							nodekey.PublicKey.Curve = crypto.S256()
							nodekey.D = new(big.Int)
							nodekey.D.SetBytes(kp.D) 
							nodekey.PublicKey.X = new(big.Int)
							nodekey.PublicKey.X.SetBytes(kp.X)
							nodekey.PublicKey.Y = new(big.Int)
							nodekey.PublicKey.Y.SetBytes(kp.Y)
							config.key = nodekey
						}
					} else {
						return NewConfigError(ERR_INVALID_SECRET_FILE, err.Error())
					}
				} else {
					// generate new secret key and persist to file
					nodekey, _ := crypto.GenerateKey()
					kp := keyPair {
						Curve: "S256",
						X: nodekey.X.Bytes(),
						Y: nodekey.Y.Bytes(),
						D: nodekey.D.Bytes(),
					}
					if data, err := json.Marshal(kp); err == nil {
						if file, err := os.Create(*params.KeyFile); err == nil {
							file.Write(data)
						} else {
							return NewConfigError(ERR_INVALID_SECRET_FILE, err.Error())
						}
					} else {
						return NewConfigError(ERR_MARSHAL_FAILURE, err.Error())
					}
					config.key = nodekey
				}
				c = &config
				return nil
			}
		} else {
			return NewConfigError(ERR_NULL_CONFIG, err.Error())
		}
	} else {
		return NewConfigError(ERR_INVALID_FILE, err.Error())
	}
}