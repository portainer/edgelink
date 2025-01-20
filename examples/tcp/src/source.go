package main

import (
	"log"

	"github.com/portainer/edgelink"
	"github.com/portainer/edgelink/examples/tcp"
)

const (
	destinationPublicIP = "192.168.1.100"
	tcpPort             = 9999
)

func main() {
	node, err := edgelink.NewNode()
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

	virtualNetwork := node.GetVirtualNetwork()

	err = tcp.TestSendAndReceiveOverTCP("10.0.0.2", tcpPort, "10.0.0.1", tcpPort, virtualNetwork)
	if err != nil {
		log.Fatalf("Failed to test TCP communication: %v", err)
	}

	log.Println("Source node is running")
	select {}
}
