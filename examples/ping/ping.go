package ping

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/netip"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// TestPingIPv4 sends an ICMP echo request and checks the response.
func TestPingIPv4(localIP, remoteIP string, mtu int, timeout time.Duration, vNet *netstack.Net) error {
	pingConn, err := setupPingConnection(localIP, remoteIP, timeout, vNet)
	if err != nil {
		return err
	}
	defer pingConn.Close()

	icmpMessage, err := createICMPEchoRequest()
	if err != nil {
		return err
	}

	if err := sendPing(pingConn, icmpMessage); err != nil {
		return err
	}

	return receivePingResponse(pingConn, icmpMessage, mtu)
}

// setupPingConnection initializes the ping connection.
func setupPingConnection(localIP, remoteIP string, timeout time.Duration, vNet *netstack.Net) (*netstack.PingConn, error) {
	localAddr, err := netip.ParseAddr(localIP)
	if err != nil {
		return nil, fmt.Errorf("unable to parse local IP address %s: %w", localIP, err)
	}

	remoteAddr, err := netip.ParseAddr(remoteIP)
	if err != nil {
		return nil, fmt.Errorf("unable to parse remote IP address %s: %w", remoteIP, err)
	}

	pingConn, err := vNet.DialPingAddr(localAddr, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("unable to create ping connection: %w", err)
	}

	pingConn.SetReadDeadline(time.Now().Add(timeout))
	return pingConn, nil
}

// createICMPEchoRequest constructs an ICMP echo request message.
func createICMPEchoRequest() (*icmp.Message, error) {
	return &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   rand.Intn(1 << 16),
			Seq:  rand.Intn(1 << 16),
			Data: []byte("HELLO-WORLD"),
		},
	}, nil
}

// sendPing sends the ICMP echo request.
func sendPing(pingConn *netstack.PingConn, icmpMessage *icmp.Message) error {
	icmpPacket, err := icmpMessage.Marshal(nil)
	if err != nil {
		return fmt.Errorf("unable to marshal ICMP message: %w", err)
	}

	_, err = pingConn.Write(icmpPacket)
	if err != nil {
		return fmt.Errorf("unable to write ICMP message: %w", err)
	}

	return nil
}

// receivePingResponse reads and validates the ICMP echo reply.
func receivePingResponse(pingConn *netstack.PingConn, icmpMessage *icmp.Message, mtu int) error {
	response := make([]byte, mtu)

	n, err := pingConn.Read(response)
	if err != nil {
		return fmt.Errorf("unable to read ICMP message: %w", err)
	}

	replyMessage, err := icmp.ParseMessage(1, response[:n])
	if err != nil {
		return fmt.Errorf("unable to parse ICMP message: %w", err)
	}

	replyEcho, ok := replyMessage.Body.(*icmp.Echo)
	if !ok {
		return fmt.Errorf("invalid reply type: %v", replyMessage)
	}

	if !bytes.Equal(replyEcho.Data, []byte("HELLO-WORLD")) || replyEcho.Seq != icmpMessage.Body.(*icmp.Echo).Seq {
		return fmt.Errorf("invalid ping reply: %v", replyEcho)
	}

	log.Printf("Ping latency: %v", time.Since(time.Now()))
	return nil
}
