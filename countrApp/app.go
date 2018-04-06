package main

import (
	"bufio"
	"strconv"
	"flag"
	"fmt"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/config"
	"github.com/trust-net/go-trust-net/common"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/network"
	"os"
	"strings"
)

type KeyPair struct {
	Curve string
	X, Y []byte
	D []byte
}

const cmdPrompt = "Command: "

var myId = &core.Byte64{}
//var peers = make(map[string]*network.AppConfig)
// define a transaction payload structure
type testTx struct {
	Op string
	Target string
	Delta int64
}

func incrementTx(name string, delta int) []byte {
	tx := testTx{"incr", name, int64(delta)}
	txPayload, _ := common.Serialize(tx)
	return txPayload
}

func decrementTx(name string, delta int) []byte {
	tx := testTx{"decr", name, int64(delta)}
	txPayload, _ := common.Serialize(tx)
	return txPayload
}

func CLI(c chan int, counterMgr network.PlatformManager) {
	config, _ := config.Config()
	for {
		fmt.Printf(cmdPrompt)
		defer func() { c <- 1 }()
		lineScanner := bufio.NewScanner(os.Stdin)
		for lineScanner.Scan() {
			line := lineScanner.Text()
			if len(line) != 0 {
				wordScanner := bufio.NewScanner(strings.NewReader(line))
				wordScanner.Split(bufio.ScanWords)
				for wordScanner.Scan() {
					cmd := wordScanner.Text()
					switch cmd {
					case "quit":
						fallthrough
					case "q":
						return
					case "log":
						wordScanner.Scan()
						if level := wordScanner.Text(); len(level) == 0 {
							fmt.Printf("usage: log <DEBUG|INFO|ERROR>\n")
						} else {
							switch level {
								case "DEBUG":
									log.SetLogLevel(log.DEBUG)
								case "INFO":
									log.SetLogLevel(log.INFO)
								case "ERROR":
									log.SetLogLevel(log.ERROR)
								default:
									fmt.Printf("Invalid log level '%s'\n", level)
							}
						}
					case "countr":
						wordScanner.Scan()
						if name := wordScanner.Text(); len(name) == 0 {
							fmt.Printf("usage: countr <countr name>\n")
						} else {
							// get current network counter value
							val, _ := counterMgr.State().Get([]byte(name))
							fmt.Printf("%d", int64(core.BytesToByte8(val).Uint64()))
						}
					case "incr":
						wordScanner.Scan()
						if name := wordScanner.Text(); len(name) == 0 {
							fmt.Printf("usage: incr <countr name> [<integer>]\n")
						} else {
							wordScanner.Scan()
							delta := 1
							var err error
							if word := wordScanner.Text(); len(word) != 0 {
								if delta, err = strconv.Atoi(word); err != nil {
									fmt.Printf("usage: incr <countr name> [<integer>]\n")
								}
							}
							// submit a transaction to increment counter
							if err == nil {
								counterMgr.Submit(incrementTx(name, delta), myId)
							}
						}
					case "decr":
						wordScanner.Scan()
						if name := wordScanner.Text(); len(name) == 0 {
							fmt.Printf("usage: decr <countr name> [<integer>]\n")
						} else {
							wordScanner.Scan()
							delta := 1
							var err error
							if word := wordScanner.Text(); len(word) != 0 {
								if delta, err = strconv.Atoi(word); err != nil {
									fmt.Printf("usage: decr <countr name> [<integer>]\n")
								}
							}
							// submit a transaction to decrement counter
							if err == nil {
								counterMgr.Submit(decrementTx(name, delta), myId)
							}
						}
					case "peers":
						for i, peer := range counterMgr.Peers() {
//						i := 0
//						for _, peer := range peers {
//							i++
							fmt.Printf("%02d : \"% 10s\" : %s\n", i, peer.NodeName, peer.NodeId)
						}
					case "info":
						fmt.Printf("#######################\n")
						fmt.Printf("Node Name: %s\n", *config.NodeName())
						fmt.Printf("Node ID  : %s\n", *config.Id())
						fmt.Printf("Network  : %s\n", *config.NetworkId())
						fmt.Printf("#######################")
					default:
						fmt.Printf("Unknown Command: %s", cmd)
						for wordScanner.Scan() {
							fmt.Printf(" %s", wordScanner.Text())
						}
						break
					}
				}
			}
			fmt.Printf("\n%s", cmdPrompt)
		}
	}
}

func main() {
	port := flag.String("port", "30303", "port to listen on")
	fileName := flag.String("file", "", "config file name")
	natEnabled := flag.Bool("nat", false, "enable NAT translation")
	flag.Parse()
	if err := config.InitializeConfig(fileName, port, natEnabled); err != nil {
		fmt.Printf("Failed to initialize configuration: %s\n", err)
		return
	}
	config, _ := config.Config()
	fmt.Printf("Starting node, listening on %s...\n", *config.Port())
	log.SetLogLevel(log.DEBUG)
	// build a transaction processor using above defined payload
	txProcessor := func(txs *network.Transaction) bool{
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
				 	fmt.Printf("Unknown opcode: %s\n%s", opCode.Op,cmdPrompt)
				 	return false
			}
//			fmt.Printf("%s: %d\n%s", opCode.Target, currVal, cmdPrompt)
			return txs.Update([]byte(opCode.Target), core.Uint64ToByte8(uint64(currVal)).Bytes())
		}
	}
	// define a pow approver for application
	powApprover := func(hash []byte, blockTs, parentTs uint64) bool {
//		// make sure first 4 bytes are 0x00
//		return hash[0] == 0x00 && hash[1] == 0x00 && hash[3] == 0x00 && hash[4] == 0x00
		td := 2
		result := true
		for i := 0; i < td; i++ {
			result = result && hash[i] == 0x00
		}
		return result
	}
	conf := network.PlatformConfig {
		AppConfig: network.AppConfig {
			NetworkId: *core.BytesToByte16([]byte(*config.NetworkId())),
			NodeName: *config.NodeName(),
		},
		ServiceConfig: network.ServiceConfig{
			IdentityKey: config.Key(),
			Port: *port,
			ProtocolName: "countr",
			ProtocolVersion: 0x01,
			TxProcessor: txProcessor,
			BootstrapNodes: config.BootnodeStrings(),
			PowApprover: powApprover,
			PeerValidator: func(config *network.AppConfig) error {
//				peers[config.NodeId] = config
				fmt.Printf("Connection request from new peer: %s\n%s", config.NodeName,cmdPrompt)
				return nil
			},
		},
	}
	if counterMgr, err := network.NewPlatformManager(&conf.AppConfig, &conf.ServiceConfig, config.Db()); err != nil {
		fmt.Printf("Failed to instantiate platform layer: %s\n", err)
		return
	} else {
		counterMgr.Start()
		c := make(chan int)
		log.AppLogger().Info("starting CLI now...")
		go CLI(c, counterMgr)
		<-c
		log.AppLogger().Info("done.")
		counterMgr.Stop()
	}
}