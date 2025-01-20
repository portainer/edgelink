package main

import (
	"log"

	"github.com/portainer/edgelink"
	"github.com/portainer/edgelink/examples/tcp"
)

const (
	tcpPort = 9999
)

func main() {
	node, err := edgelink.NewNode()
	if err != nil {
		log.Fatalf("Failed to create edgelink node: %v", err)
	}

	err = node.SetupAsDestination()
	if err != nil {
		log.Fatalf("Failed to setup node as destination: %v", err)
	}

	err = node.Link()
	if err != nil {
		log.Fatalf("Unable to establish link with source node: %v", err)
	}

	virtualNetwork := node.GetVirtualNetwork()

	err = tcp.TestReceiveAndSendOverTCP("10.0.0.1", tcpPort, "10.0.0.2", tcpPort, virtualNetwork)
	if err != nil {
		log.Fatalf("Failed to test TCP communication: %v", err)
	}

	log.Println("Destination node is running")
	select {}
}
