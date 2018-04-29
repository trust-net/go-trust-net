package main

import (
	"bufio"
	"strconv"
	"flag"
	"time"
	"fmt"
	"encoding/hex"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/config"
	"github.com/trust-net/go-trust-net/consensus"
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

var myId []byte
//var peers = make(map[string]*network.AppConfig)
// define a transaction payload structure
type testTx struct {
	Op string
	Source string
	Target string
	Delta int64
}

func incrementTx(name string, delta int) []byte {
	tx := testTx{
		Op: "incr",
		Target: name,
		Delta: int64(delta),
	}
	txPayload, _ := common.Serialize(tx)
	return txPayload
}

func decrementTx(name string, delta int) []byte {
	tx := testTx{
		Op: "decr",
		Target: name,
		Delta: int64(delta),
	}
	txPayload, _ := common.Serialize(tx)
	return txPayload
}

type op struct {
	name string
	delta int
}

func scanOps(scanner *bufio.Scanner) (ops []op) {
	nextToken := func() (*string, int, bool) {
		if !scanner.Scan() {
			return nil, 0, false
		}
		word := scanner.Text()
		if delta, err := strconv.Atoi(word); err == nil {
			return nil, delta, true
		} else {
			return &word, 0, true
		}
	}
	ops = make([]op, 0)
	currOp := op {}
	readName := false
	for {
		name, delta, success := nextToken()

		if !success {
			if readName {
				ops = append(ops, currOp)
			}
			return
		} else if name == nil && currOp.name == "" {
			return
		}

		if name != nil {
			if readName {
				ops = append(ops, currOp)
			}
			currOp = op {}
			currOp.name = *name
			currOp.delta = 1
			readName = true
		} else {
			currOp.delta = delta
			ops = append(ops, currOp)
			currOp = op {}
			readName = false
		}
	}
}

func CLI(c chan int, counterMgr network.PlatformManager) {
	config, _ := config.Config()
	defer func() { c <- 1 }()
	for {
		fmt.Printf(cmdPrompt)
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
						hasNext := wordScanner.Scan()
						oneDone := false
						for hasNext {
							name := wordScanner.Text()
							if len(name) != 0 {
								if oneDone {
									fmt.Printf("\n")
								} else {
									oneDone = true
								}
								// get current network counter value
								val, _ := counterMgr.State().Get([]byte(name))
								fmt.Printf("% 10s: %d", name, int64(core.BytesToByte8(val).Uint64()))
							}
							hasNext = wordScanner.Scan()
						}
						if !oneDone {
							fmt.Printf("usage: countr <countr name> ...\n")
						}
					case "incr":
						ops := scanOps(wordScanner)
						if len(ops) == 0 {
							fmt.Printf("usage: incr <countr name> [<integer>] ...\n")
						} else {
							for _, op := range ops {
								fmt.Printf("adding transaction: incr %s %d\nTX ID: %x\n", op.name, op.delta,
									*counterMgr.Submit(incrementTx(op.name, op.delta), nil, myId))
							}
						}
					case "decr":
						ops := scanOps(wordScanner)
						if len(ops) == 0 {
							fmt.Printf("usage: decr <countr name> [<integer>] ...\n")
						} else {
							for _, op := range ops {
								fmt.Printf("adding transaction: decr %s %d\nTX ID: %x\n", op.name, op.delta,
									*counterMgr.Submit(decrementTx(op.name, op.delta), nil, myId))
							}
						}
					case "peers":
						for _, peer := range counterMgr.Peers() {
//						i := 0
//						for _, peer := range peers {
//							i++
							fmt.Printf("% 10s\" [%s RTU] : %x\n", "\"" + peer.NodeName, counterMgr.MiningRewardBalance(peer.NodeId), peer.NodeId)
						}
					case "tx":
						wordScanner.Scan()
						if tx := wordScanner.Text(); len(tx) == 0 {
							fmt.Printf("usage: tx <transaction id>\n")
						} else {
							if bytes, err := hex.DecodeString(tx); err == nil {
								if block, err := counterMgr.Status(core.BytesToByte64(bytes)); err == nil {
									t := time.Unix(0,int64(block.Timestamp().Uint64()))
									fmt.Printf("Transaction accepted at depth %d @ %s\n", block.Depth().Uint64(), t.Format("Mon Jan 2 15:04:05 UTC 2006"))
								} else {
									switch err.(*core.CoreError).Code() {
										case consensus.ERR_TX_NOT_APPLIED:
											if block == nil {
												fmt.Printf("Transaction rejected\n")
											} else {
												fmt.Printf("Transaction not in canonical chain\n")
											}
										case consensus.ERR_TX_NOT_FOUND:
											fmt.Printf("Transaction not submitted\n")
									}
								}
							} else {
								fmt.Printf("Invalid tx: %s\n", err)
							}
						}
					case "balance":
						wordScanner.Scan()
						if account := wordScanner.Text(); len(account) == 0 {
							fmt.Printf("usage: balance <account number>\n")
						} else {
							if bytes, err := hex.DecodeString(account); err == nil {
								fmt.Printf("%s RTU\n", counterMgr.MiningRewardBalance(bytes))
							} else {
								fmt.Printf("Invalid address: %s\n", err)
							}
						}
					case "xfer":
						wordScanner.Scan()
						amount := wordScanner.Text()
						wordScanner.Scan()
						account := wordScanner.Text()
						if len(amount) == 0 || len(account) == 0{
							fmt.Printf("usage: xfer <amount> <account number>\n")
						} else {
							delta, _ := strconv.ParseInt(amount, 10, 64)
							fmt.Printf("adding transaction: xfer %d --> %s\n", delta, account)
							txPayload, _ := common.Serialize(testTx{
								Op: "xfer",
								Source: fmt.Sprintf("%x", config.Id()),
								Target: account,
								Delta: delta,
							})
							counterMgr.Submit(txPayload, nil, myId)
						}
					case "info":
						state := counterMgr.State()
						t := time.Unix(0,int64(state.Timestamp()))
						fmt.Printf("#######################\n")
						fmt.Printf("Node Name: %s\n", *config.NodeName())
						fmt.Printf("Node ID  : %x\n", config.Id())
						fmt.Printf("Reward   : %s RTU\n", counterMgr.MiningRewardBalance(nil))
						fmt.Printf("Network  : %s\n", *config.NetworkId())
						fmt.Printf("TIP      : %x\n", state.Tip())
						fmt.Printf("Depth    : %d\n", state.Depth())
						fmt.Printf("Weight   : %d\n", state.Weight())
						fmt.Printf("Timestamp: %s\n", t.Format("Mon Jan 2 15:04:05 UTC 2006"))
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

func lookup (txs *network.Transaction, key string) int64 {
	currVal := int64(0)
	if val, err := txs.Lookup([]byte(key)); err == nil {
		currVal = int64(core.BytesToByte8(val).Uint64())
	}
	return currVal
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
	myId = config.Id()
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
			switch opCode.Op {
				case "incr":
					currVal = lookup(txs, opCode.Target) + opCode.Delta
				case "decr":
					currVal = lookup(txs, opCode.Target) - opCode.Delta
				case "xfer":
					// We are NOT doing signed transactions, and hence xfer cannot be authenticated!!!
					amount := network.Uint64ToRtu(uint64(opCode.Delta))
					var err error
					if err = network.Debit(txs.Block(), []byte(opCode.Source), amount); err != nil {
						fmt.Printf("%s: %s <--debit--- %s\n%s", err, amount, opCode.Source, cmdPrompt)
						return false
					}
					if err = network.Credit(txs.Block(), []byte(opCode.Target), amount); err != nil {
						fmt.Printf("Failed to credit: %s\n%s", err, cmdPrompt)
						return false
					}
					return true
			 	default:
				 	fmt.Printf("Unknown opcode: %s\n%s", opCode.Op,cmdPrompt)
				 	return false
			}
			return txs.Update([]byte(opCode.Target), core.Uint64ToByte8(uint64(currVal)).Bytes())
		}
	}
	// define a pow approver for application
	powApprover := func(hash []byte, blockTs, parentTs uint64) bool {
		// make sure first n bytes are 0x00
		n := 2
		result := true
		for i := 0; i < n; i++ {
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
