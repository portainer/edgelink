package main

import (
	"log"

	"github.com/portainer/edgelink"
	"github.com/portainer/edgelink/examples/grpc"
)

const (
	destinationPublicIP = "192.168.1.100"
	grpcPort            = 10777
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

	err = grpc.TestClient("10.0.0.1", grpcPort, virtualNetwork)
	if err != nil {
		log.Fatalf("Failed to communicate with destination node over gRPC: %v", err)
	}
}
