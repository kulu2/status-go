// bootnode runs a bootstrap node for the Ethereum Discovery Protocol.
package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

type bootnodes []*discv5.Node

func (f *bootnodes) String() string {
	return "discv5 nodes"
}

// Set unmarshals enode into discv5.Node.
func (f *bootnodes) Set(value string) error {
	n, err := discv5.ParseNode(value)
	if err != nil {
		return err
	}
	*f = append(*f, n)
	return nil
}

func main() {
	var (
		listenAddr  = flag.String("addr", ":30301", "listen address")
		nodeKeyFile = flag.String("nodekey", "", "private key filename")
		keydata     = flag.String("keydata", "", "hex encoded private key")
		verbosity   = flag.Int("verbosity", int(log.LvlInfo), "log verbosity (0-9)")
		vmodule     = flag.String("vmodule", "", "log verbosity pattern")
		nursery     = bootnodes{}
	)
	flag.Var(&nursery, "n", "These nodes are used to connect to the network if the table is empty and there are no known nodes in the database.")
	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*verbosity))
	if err := glogger.Vmodule(*vmodule); err != nil {
		log.Crit("Failed to set glog verbosity", "value", *vmodule, "err", err)
	}
	log.Root().SetHandler(glogger)

	if len(*nodeKeyFile) == 0 && len(*keydata) == 0 {
		log.Crit("either `nodekey` or `keydata` must be provided")
	}
	var (
		nodeKey *ecdsa.PrivateKey
		err     error
	)
	if len(*nodeKeyFile) != 0 {
		nodeKey, err = crypto.LoadECDSA(*nodeKeyFile)
		if err != nil {
			log.Crit("Failed to load ecdsa key from", "file", *nodeKeyFile, "error", err)
		}
	} else if len(*keydata) != 0 {
		log.Warn("key will be visible in process list. should be used only for tests")
		key, err := hex.DecodeString(*keydata)
		if err != nil {
			log.Crit("unable to decode hex", "data", keydata, "error", err)
		}
		nodeKey, err = crypto.ToECDSA(key)
		if err != nil {
			log.Crit("unable to convert decoded hex into ecdsa.PrivateKey", "data", key, "error", err)
		}
	}

	addr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		log.Crit("Unable to resolve UDP", "address", *listenAddr, "error", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Crit("Unable to listen on udp", "address", addr, "error", err)
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	tab, err := discv5.ListenUDP(nodeKey, conn, realaddr, "", nil)
	if err != nil {
		log.Crit("Failed to create discovery v5 table:", "error", err)
	}
	defer tab.Close()
	if err := tab.SetFallbackNodes(nursery); err != nil {
		log.Crit("Failed to set fallback", "nodes", nursery, "error", err)
	}
	select {}
}
