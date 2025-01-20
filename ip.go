package edgelink

import (
	"fmt"
	"net/netip"
)

func parseDNSAddresses(dnsList []string) ([]netip.Addr, error) {
	nameservers := make([]netip.Addr, len(dnsList))
	for i, dns := range dnsList {
		addr, err := netip.ParseAddr(dns)
		if err != nil {
			return nil, fmt.Errorf("invalid DNS address: %w", err)
		}
		nameservers[i] = addr
	}
	return nameservers, nil
}

// buildAllowedIP constructs an allowed IP range for a given IP address
// using a /30 subnet mask. In the context of edgelink, this function
// is crucial for defining the range of IP addresses that are permitted
// to communicate with a node. The /30 subnet mask allows for four IP
// addresses, typically used to create a small point-to-point network
// between two nodes.
//
// In a /30 subnet, the four IP addresses are allocated as follows:
// - The first IP address is the network address and cannot be assigned to a device.
// - The second IP address can be assigned to one node.
// - The third IP address can be assigned to the other node.
// - The fourth IP address is the broadcast address and cannot be assigned to a device.
func buildAllowedIP(ip string) string {
	return fmt.Sprintf("%s/30", ip)
}
