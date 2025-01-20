package main

import (
	"log"

	"github.com/portainer/edgelink"
	"github.com/portainer/edgelink/examples/grpc"
)

const (
	grpcPort = 10777
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

	err = grpc.TestServer("10.0.0.1", grpcPort, virtualNetwork)
	if err != nil {
		log.Fatalf("Failed to setup commcheck server: %v", err)
	}
}
