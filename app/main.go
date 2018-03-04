package main

import (
	"bufio"
	"strconv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/trust-net/go-trust-net/log"
	"github.com/trust-net/go-trust-net/config"
	"github.com/trust-net/go-trust-net/core"
	"github.com/trust-net/go-trust-net/protocol/pager"
	"github.com/trust-net/go-trust-net/protocol/counter"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/satori/go.uuid"
	"os"
	"strings"
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
							MsgId:	 *core.BytesToByte16(id.Bytes()),
						}
						log.AppLogger().Debug("sent message: '%s' to %d peers", text, pagerMgr.Broadcast(msg))
					case "countr":
						// flush the input line
						for wordScanner.Scan() {
							wordScanner.Text()
						}
						// get current network counter value
						fmt.Printf("%d", counterMgr.Countr())
					case "incr":
						wordScanner.Scan()
						delta := 1
						var err error
						if word := wordScanner.Text(); len(word) != 0 {
							if delta, err = strconv.Atoi(word); err != nil {
								fmt.Printf("usage: incr <integer>\n")
							}
						}
						// ask counter manager to increment counter
						if err == nil {
							counterMgr.Increment(delta)
						}
					case "decr":
						wordScanner.Scan()
						delta := 1
						var err error
						if word := wordScanner.Text(); len(word) != 0 {
							if delta, err = strconv.Atoi(word); err != nil {
								fmt.Printf("usage: decr <integer>\n")
							}
						}
						// ask counter manager to increment counter
						if err == nil {
							counterMgr.Decrement(delta)
						}
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
						fmt.Printf("#######################\n")
						fmt.Printf("Node Name: %s\n", *config.NodeName())
						fmt.Printf("Node ID  : %s\n", *config.Id())
						fmt.Printf("Network  : %s\n", *config.NetworkId())
//						fmt.Printf("%s", srv.NodeInfo())
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
	
	pagerMgr = pager.NewPagerProtocolManager(func(from, text string){
		fmt.Printf("\n########## Msg From '%s' #########\n", from)
		fmt.Printf("%s\n#######################\n%s", text, cmdPrompt)

	})
	defer pagerMgr.Shutdown()
	counterMgr = counter.NewCountrProtocolManager(*config.Id())
	defer counterMgr.Shutdown()
	protocols := make([]p2p.Protocol,0)
	protocols = append(protocols, pagerMgr.Protocol())
	protocols = append(protocols, counterMgr.Protocol())
	nat := nat.Any()
	if !*config.NatEnabled() {
		nat = nil
	}
	serverConfig := p2p.Config{
		MaxPeers:   10,
		PrivateKey: config.Key(),
		Name:       *config.NodeName(),
		ListenAddr: ":" + *config.Port(),
		NAT: 		nat,
		Protocols:  protocols,
		BootstrapNodes: config.Bootnodes(),
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
		srv.Stop()
	}
}
