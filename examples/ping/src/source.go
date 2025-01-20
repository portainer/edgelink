package main

import (
	"log"
	"time"

	"github.com/portainer/edgelink"
	"github.com/portainer/edgelink/examples/ping"
)

const (
	pingInterval        = 1 * time.Second
	pingTimeout         = 3 * time.Second
	mtu                 = 1500
	destinationPublicIP = "192.168.1.100"
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

	for i := 0; i < 3; i++ {
		err = ping.TestPingIPv4("10.0.0.2", "10.0.0.1", mtu, pingTimeout, virtualNetwork)
		if err != nil {
			log.Printf("Failed to ping destination node on attempt %d: %v", i+1, err)
		}

		time.Sleep(pingInterval)
	}

	log.Println("Source node is running")
	select {}
}
