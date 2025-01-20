package main

import (
	"flag"
	"log"

	"github.com/portainer/edgelink"
)

const (
	destinationPublicIP = "192.168.1.100"
)

func main() {
	privateKey := flag.String("private-key", "", "Private key for the node")
	peerPublicKey := flag.String("peer-public-key", "", "Public key of the peer node")
	flag.Parse()

	if *privateKey == "" || *peerPublicKey == "" {
		log.Fatal("Both private-key and peer-public-key must be provided")
	}

	node, err := edgelink.NewNode(
		edgelink.WithPrivateKey(*privateKey),
		edgelink.WithPeerPublicKey(*peerPublicKey),
	)
	if err != nil {
		log.Fatalf("Failed to create edgelink node: %v", err)
	}

	err = node.SetupAsSource(destinationPublicIP)
	if err != nil {
		log.Fatalf("Failed to setup node as source: %v", err)
	}

	err = node.Link()
	if err != nil {
		log.Fatalf("Unable to establish link with source node: %v", err)
	}

	_ = node.GetVirtualNetwork()

	// Use the virtual network interface to communicate with the destination node

	log.Println("Source node is running")
	select {}
}
