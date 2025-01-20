package edgelink

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"net/netip"

	"github.com/portainer/edgelink/wireguard"
)

func TestSetupAsSource(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		node := &Node{}
		destinationPublicIP := "192.168.1.100"

		err := node.SetupAsSource(destinationPublicIP)
		if err != nil {
			t.Fatalf("SetupAsSource() error = %v, want nil", err)
		}

		if node.source == nil {
			t.Fatal("SetupAsSource() source is not set")
		}

		if node.source.destinationPublicIP != destinationPublicIP {
			t.Errorf("destinationPublicIP = %v, want %v", node.source.destinationPublicIP, destinationPublicIP)
		}

		if node.source.keyxPort != keyxDefaultPort {
			t.Errorf("keyxPort = %v, want %v", node.source.keyxPort, keyxDefaultPort)
		}

		if node.source.wgPort != wgDefaultPort {
			t.Errorf("wgPort = %v, want %v", node.source.wgPort, wgDefaultPort)
		}

		if node.source.localIP != wgDefaultSourceLocalIP {
			t.Errorf("localIP = %v, want %v", node.source.localIP, wgDefaultSourceLocalIP)
		}

		if node.source.destinationIP != wgDefaultDestinationLocalIP {
			t.Errorf("destinationIP = %v, want %v", node.source.destinationIP, wgDefaultDestinationLocalIP)
		}

		if node.source.wgVerboseLogging != false {
			t.Errorf("wgVerboseLogging = %v, want %v", node.source.wgVerboseLogging, false)
		}
	})

	t.Run("AllOptions", func(t *testing.T) {
		node := &Node{}
		destinationPublicIP := "192.168.1.100"
		localIP := "192.168.1.1"
		destinationIP := "192.168.1.2"
		keyxPort := 51878
		keyxRetryInterval := 1 * time.Second
		keyxTimeout := 1 * time.Second
		wgPort := 51877
		wgVerboseLogging := true

		err := node.SetupAsSource(destinationPublicIP,
			WithSourceLocalIP(localIP),
			WithSourceTargetIP(destinationIP),
			WithSourceWGPort(wgPort),
			WithSourceKeyxPort(keyxPort),
			WithSourceKeyxRetryInterval(keyxRetryInterval),
			WithSourceKeyxTimeout(keyxTimeout),
			WithSourceWGVerboseLogging(wgVerboseLogging),
		)
		if err != nil {
			t.Fatalf("SetupAsSource() error = %v, want nil", err)
		}

		if node.source.localIP != localIP {
			t.Errorf("localIP = %v, want %v", node.source.localIP, localIP)
		}

		if node.source.destinationIP != destinationIP {
			t.Errorf("destinationIP = %v, want %v", node.source.destinationIP, destinationIP)
		}

		if node.source.wgPort != wgPort {
			t.Errorf("wgPort = %v, want %v", node.source.wgPort, wgPort)
		}

		if node.source.keyxPort != keyxPort {
			t.Errorf("keyxPort = %v, want %v", node.source.keyxPort, keyxPort)
		}

		if node.source.wgVerboseLogging != wgVerboseLogging {
			t.Errorf("wgVerboseLogging = %v, want %v", node.source.wgVerboseLogging, wgVerboseLogging)
		}
	})

	t.Run("MissingDestinationPublicIP", func(t *testing.T) {
		node := &Node{}

		err := node.SetupAsSource("")
		if err == nil {
			t.Error("SetupAsSource() expected error for missing destination public IP, got nil")
		}
	})

	t.Run("InvalidLocalIP", func(t *testing.T) {
		node := &Node{}
		destinationPublicIP := "192.168.1.100"

		err := node.SetupAsSource(destinationPublicIP, WithSourceLocalIP("invalid-ip"))
		if err == nil {
			t.Error("SetupAsSource() expected error for invalid local IP, got nil")
		}
	})

	t.Run("InvalidDestinationIP", func(t *testing.T) {
		node := &Node{}
		destinationPublicIP := "192.168.1.100"

		err := node.SetupAsSource(destinationPublicIP, WithSourceTargetIP("invalid-ip"))
		if err == nil {
			t.Error("SetupAsSource() expected error for invalid destination IP, got nil")
		}
	})
}

func TestValidateSource(t *testing.T) {
	t.Run("ValidSource", func(t *testing.T) {
		s := &source{
			destinationPublicIP: "192.168.1.100",
			localIP:             "10.0.0.1",
			destinationIP:       "10.0.0.2",
		}

		err := validateSource(s)
		if err != nil {
			t.Errorf("validateSource() error = %v, want nil", err)
		}
	})

	t.Run("MissingDestinationPublicIP", func(t *testing.T) {
		s := &source{
			localIP:       "10.0.0.1",
			destinationIP: "10.0.0.2",
		}

		err := validateSource(s)
		if err == nil {
			t.Fatal("validateSource() expected error for missing destination public IP, got nil")
		}
	})

	t.Run("InvalidLocalIP", func(t *testing.T) {
		s := &source{
			destinationPublicIP: "192.168.1.100",
			localIP:             "invalid-ip",
			destinationIP:       "10.0.0.2",
		}

		err := validateSource(s)
		if err == nil {
			t.Fatal("validateSource() expected error for invalid local IP, got nil")
		}
	})

	t.Run("InvalidDestinationIP", func(t *testing.T) {
		s := &source{
			destinationPublicIP: "192.168.1.100",
			localIP:             "10.0.0.1",
			destinationIP:       "invalid-ip",
		}

		err := validateSource(s)
		if err == nil {
			t.Fatal("validateSource() expected error for invalid destination IP, got nil")
		}
	})
}

func TestCreateSourceWireGuardConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		node := &Node{
			config: nodeConfig{
				dns:           []string{"8.8.8.8", "8.8.4.4"},
				mtu:           1500,
				peerPublicKey: "eb9b399c7bb62903e47353f3e752a11c61741d11d8119be3272dd7d204fb9d5b",
				privateKey:    "787538cae8c33db4c368c3ef7ba5fa74950a6251076357579a3d1312e707cf48",
			},
			source: &source{
				destinationPublicIP: "192.168.1.100",
				localIP:             "10.0.0.1",
				destinationIP:       "10.0.0.2",
				wgPort:              wgDefaultPort,
				wgVerboseLogging:    false,
			},
		}

		config, err := node.createSourceWireGuardConfig()
		if err != nil {
			t.Fatalf("createSourceWireGuardConfig() error = %v, want nil", err)
		}

		expectedIP, _ := netip.ParseAddr("10.0.0.1")
		expectedDNS, _ := parseDNSAddresses([]string{"8.8.8.8", "8.8.4.4"})
		expectedEndpoint := fmt.Sprintf("%s:%d", node.source.destinationPublicIP, node.source.wgPort)

		expectedConfig := wireguard.WireGuardConfig{
			PrivateKey:     node.config.privateKey,
			PeerPublicKey:  node.config.peerPublicKey,
			MTU:            node.config.mtu,
			DNSAddresses:   expectedDNS,
			LocalAddresses: []netip.Addr{expectedIP},
			AllowedIP:      buildAllowedIP(node.source.destinationIP),
			Endpoint:       &expectedEndpoint,
			VerboseLogging: node.source.wgVerboseLogging,
		}

		if !reflect.DeepEqual(config, expectedConfig) {
			t.Errorf("createSourceWireGuardConfig() = %v, want %v", config, expectedConfig)
		}
	})

	t.Run("InvalidLocalIP", func(t *testing.T) {
		node := &Node{
			config: nodeConfig{
				privateKey:    "787538cae8c33db4c368c3ef7ba5fa74950a6251076357579a3d1312e707cf48",
				peerPublicKey: "eb9b399c7bb62903e47353f3e752a11c61741d11d8119be3272dd7d204fb9d5b",
				mtu:           1500,
				dns:           []string{"8.8.8.8", "8.8.4.4"},
			},
			source: &source{
				destinationPublicIP: "192.168.1.100",
				localIP:             "invalid-ip",
				destinationIP:       "10.0.0.2",
				wgPort:              wgDefaultPort,
				wgVerboseLogging:    false,
			},
		}

		_, err := node.createSourceWireGuardConfig()
		if err == nil {
			t.Fatal("createSourceWireGuardConfig() expected error for invalid local IP, got nil")
		}
	})

	t.Run("InvalidDNS", func(t *testing.T) {
		node := &Node{
			config: nodeConfig{
				privateKey:    "787538cae8c33db4c368c3ef7ba5fa74950a6251076357579a3d1312e707cf48",
				peerPublicKey: "eb9b399c7bb62903e47353f3e752a11c61741d11d8119be3272dd7d204fb9d5b",
				mtu:           1500,
				dns:           []string{"invalid-dns"},
			},
			source: &source{
				destinationPublicIP: "192.168.1.100",
				localIP:             "10.0.0.1",
				destinationIP:       "10.0.0.2",
				wgPort:              wgDefaultPort,
				wgVerboseLogging:    false,
			},
		}

		_, err := node.createSourceWireGuardConfig()
		if err == nil {
			t.Fatal("createSourceWireGuardConfig() expected error for invalid DNS, got nil")
		}
	})
}
