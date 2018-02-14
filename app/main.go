package main

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/protocol/pager"
	"github.com/trust-net/go-trust-net/protocol/counter"
	"github.com/trust-net/go-trust-net/protocol"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/satori/go.uuid"
	"os"
	"strings"
	"math/big"
)

type KeyPair struct {
	Curve string
	X, Y []byte
	D []byte
}

const cmdPrompt = "Command: "

var counterMgr *counter.CountrProtocolManager
var pagerMgr *pager.PagerProtocolManager

func CLI(c chan int, srv *p2p.Server) {
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
					case "send":
						text := ""
						for wordScanner.Scan() {
							if len(text) == 0 {
								text = wordScanner.Text()
							} else {
								text += " " + wordScanner.Text()
							}
						}
						id, _ := uuid.NewV1()
						msg := pager.BroadcastTextMsg{
							MsgText: text,
							MsgId:	 *protocol.BytesToByte16(id.Bytes()),
						}
						log.AppLogger().Debug("sent message: '%s' to %d peers", text, pagerMgr.Broadcast(msg))
					case "add":
						wordScanner.Scan()
						if peerNode := wordScanner.Text(); len(peerNode) == 0 {
							fmt.Printf("usage: add <peer node>\n")
						} else {
							var err = error(nil)
							if node, err := discover.ParseNode(peerNode); err == nil {
								log.AppLogger().Debug("adding peer node: '%s'", peerNode)
								err = pagerMgr.AddPeer(node)
							}
							if err != nil {
								log.AppLogger().Error("failed to add peer node: %s", err)
							}
						}
					case "save":
						wordScanner.Scan()
						if fileName := wordScanner.Text(); len(fileName) == 0 {
							fmt.Printf("usage: save <file name>\n")
						} else {
							fmt.Printf("persisting to '%s'", fileName)
							kpS := KeyPair {
								Curve: "S256",
								X: srv.PrivateKey.X.Bytes(),
								Y: srv.PrivateKey.Y.Bytes(),
								D: srv.PrivateKey.D.Bytes(),
							}
							if kp, err := json.Marshal(kpS); err == nil {
								fmt.Printf(" size: %d", len(kp))
								if file, err := os.Create(fileName); err == nil {
									file.Write(kp)
								} else {
									fmt.Printf("\nError: %s", err)
								}
							} else {
								fmt.Printf("failed to marshal: %s", err)
							}
						}
					case "peers":
						for i, peer := range srv.PeersInfo() {
							fmt.Printf("%02d : \"% 10s\" : %s@%s\n", i+1, peer.Name, peer.ID, peer.Network.RemoteAddress)
						}
					case "enode":
						fmt.Printf("%s", srv.NodeInfo().Enode)
					case "info":
						fmt.Printf("%s", srv.NodeInfo())
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
	bootNode := flag.String("bootnode", "", "bootstrap node address")
	name := flag.String("name", "", "name for the node")
	fileName := flag.String("file", "", "name for the file with saved identity")
	natEnabled := flag.Bool("nat", false, "enable NAT translation")
	flag.Parse()
	fmt.Printf("Starting node, listening on %s...\n", *port)
	if len(*name) == 0 {
		*name = "Node@" + *port
	}
	log.SetLogLevel(log.DEBUG)
	portS := ":" + (*port)
	var nodekey *ecdsa.PrivateKey
	if len(*fileName) == 0 {
		nodekey, _ = crypto.GenerateKey()
	} else {
		if file, err := os.Open(*fileName); err == nil {
			kp := make([]byte, 1024)
			if count, err := file.Read(kp); err == nil && count <= 1024 {
				kp = kp[:count]
				// do not log this in production, key is secret data!!!
				// log.AppLogger().Debug("Read %d bytes: %s", count, kp)
				kpS := KeyPair{}
				if err := json.Unmarshal(kp, &kpS); err != nil {
					log.AppLogger().Error("JSON Unmarshal Error: %s", err)
				} else {
					nodekey = new(ecdsa.PrivateKey)
					nodekey.PublicKey.Curve = crypto.S256()
					nodekey.D = new(big.Int)
					nodekey.D.SetBytes(kpS.D) 
					nodekey.PublicKey.X = new(big.Int)
					nodekey.PublicKey.X.SetBytes(kpS.X)
					nodekey.PublicKey.Y = new(big.Int)
					nodekey.PublicKey.Y.SetBytes(kpS.Y)
				}
			} else {
				log.AppLogger().Error("File Read Error: %s", err)
			}
		} else {
			log.AppLogger().Error("\nFile Open Error: %s\n", err)
		}
	}
	nat := nat.Any()
	if !*natEnabled {
		nat = nil
	}
	
	pagerMgr = pager.NewPagerProtocolManager(func(from, text string){
		fmt.Printf("\n########## Msg From '%s' #########\n", from)
		fmt.Printf("%s\n#######################\n%s", text, cmdPrompt)

	})
	counterMgr = counter.NewCountrProtocolManager()
	protocols := make([]p2p.Protocol,0)
	protocols = append(protocols, pagerMgr.Protocol())
	protocols = append(protocols, counterMgr.Protocol())
	
	serverConfig := p2p.Config{
		MaxPeers:   10,
		PrivateKey: nodekey,
		Name:       *name,
		ListenAddr: portS,
		NAT: 		nat,
		Protocols:  protocols,
	}
	bootNodes := make([]*discover.Node, 1)
	if len(*bootNode) == 0 {
		log.AppLogger().Info("Running without discovery")
	} else {
		log.AppLogger().Info("using '%s' for discovery", *bootNode)
		if bootStrapNode, err := discover.ParseNode(*bootNode); err == nil {
			bootNodes[0] = bootStrapNode
			serverConfig.BootstrapNodes = bootNodes
		} else {
			log.AppLogger().Error("failed to parse boot node: %s", err)
			return
		}
	}
	srv := &p2p.Server{Config: serverConfig}
	if err := srv.Start(); err != nil {
		log.AppLogger().Error("Failed to start server: %s", err)
	} else {
		log.AppLogger().Info("started server: %s", srv.NodeInfo().Enode)
		c := make(chan int)
		log.AppLogger().Info("starting CLI now...")
		go CLI(c, srv)
		<-c
		log.AppLogger().Info("done.")
//		counterMgr.Stop()
		srv.Stop()
	}
}
