package edgelink

import (
	"reflect"
	"testing"

	"net/netip"

	"github.com/portainer/edgelink/wireguard"
)

func TestSetupAsDestination(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		node := &Node{}

		err := node.SetupAsDestination()
		if err != nil {
			t.Fatalf("SetupAsDestination() error = %v, want nil", err)
		}

		if node.destination == nil {
			t.Fatal("SetupAsDestination() destination is not set")
		}

		if node.destination.keyxPort != keyxDefaultPort {
			t.Errorf("keyxPort = %v, want %v", node.destination.keyxPort, keyxDefaultPort)
		}

		if node.destination.wgPort != wgDefaultPort {
			t.Errorf("wgPort = %v, want %v", node.destination.wgPort, wgDefaultPort)
		}

		if node.destination.wgKeepAlive != wgDefaultKeepalive {
			t.Errorf("wgKeepAlive = %v, want %v", node.destination.wgKeepAlive, wgDefaultKeepalive)
		}

		if node.destination.localIP != wgDefaultDestinationLocalIP {
			t.Errorf("localIP = %v, want %v", node.destination.localIP, wgDefaultDestinationLocalIP)
		}

		if node.destination.wgVerboseLogging != false {
			t.Errorf("wgVerboseLogging = %v, want %v", node.destination.wgVerboseLogging, false)
		}

		if node.destination.sourceIP != wgDefaultSourceLocalIP {
			t.Errorf("sourceIP = %v, want %v", node.destination.sourceIP, wgDefaultSourceLocalIP)
		}
	})

	t.Run("AllOptions", func(t *testing.T) {
		node := &Node{}
		localIP := "192.168.1.1"
		sourceIP := "192.168.1.2"
		wgPort := 51820
		keyxPort := 51821
		wgKeepalive := 25
		wgVerboseLogging := true

		err := node.SetupAsDestination(
			WithDestinationLocalIP(localIP),
			WithDestinationOriginIP(sourceIP),
			WithDestinationWGPort(wgPort),
			WithDestinationKeyxPort(keyxPort),
			WithDestinationWGKeepalive(wgKeepalive),
			WithDestinationWGVerboseLogging(wgVerboseLogging),
		)
		if err != nil {
			t.Fatalf("SetupAsDestination() error = %v, want nil", err)
		}

		if node.destination.localIP != localIP {
			t.Errorf("localIP = %v, want %v", node.destination.localIP, localIP)
		}

		if node.destination.sourceIP != sourceIP {
			t.Errorf("sourceIP = %v, want %v", node.destination.sourceIP, sourceIP)
		}

		if node.destination.wgPort != wgPort {
			t.Errorf("wgPort = %v, want %v", node.destination.wgPort, wgPort)
		}

		if node.destination.keyxPort != keyxPort {
			t.Errorf("keyxPort = %v, want %v", node.destination.keyxPort, keyxPort)
		}

		if node.destination.wgKeepAlive != wgKeepalive {
			t.Errorf("wgKeepAlive = %v, want %v", node.destination.wgKeepAlive, wgKeepalive)
		}

		if node.destination.wgVerboseLogging != wgVerboseLogging {
			t.Errorf("wgVerboseLogging = %v, want %v", node.destination.wgVerboseLogging, wgVerboseLogging)
		}
	})

	t.Run("InvalidLocalIP", func(t *testing.T) {
		node := &Node{}

		err := node.SetupAsDestination(WithDestinationLocalIP("invalid-ip"))
		if err == nil {
			t.Error("SetupAsDestination() expected error for invalid local IP, got nil")
		}
	})

	t.Run("InvalidSourceIP", func(t *testing.T) {
		node := &Node{}

		err := node.SetupAsDestination(WithDestinationOriginIP("invalid-ip"))
		if err == nil {
			t.Error("SetupAsDestination() expected error for invalid source IP, got nil")
		}
	})
}

func TestValidateDestination(t *testing.T) {
	t.Run("ValidDestination", func(t *testing.T) {
		d := &destination{
			localIP:  "10.0.0.1",
			sourceIP: "10.0.0.2",
		}

		err := validateDestination(d)
		if err != nil {
			t.Errorf("validateDestination() error = %v, want nil", err)
		}
	})

	t.Run("InvalidLocalIP", func(t *testing.T) {
		d := &destination{
			localIP:  "invalid-ip",
			sourceIP: "10.0.0.2",
		}

		err := validateDestination(d)
		if err == nil {
			t.Fatal("validateDestination() expected error for invalid local IP, got nil")
		}
	})

	t.Run("InvalidSourceIP", func(t *testing.T) {
		d := &destination{
			localIP:  "10.0.0.1",
			sourceIP: "invalid-ip",
		}

		err := validateDestination(d)
		if err == nil {
			t.Fatal("validateDestination() expected error for invalid source IP, got nil")
		}
	})
}

func TestCreateDestinationWireGuardConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		node := &Node{
			config: nodeConfig{
				dns:           []string{"8.8.8.8", "8.8.4.4"},
				mtu:           1500,
				peerPublicKey: "eb9b399c7bb62903e47353f3e752a11c61741d11d8119be3272dd7d204fb9d5b",
				privateKey:    "787538cae8c33db4c368c3ef7ba5fa74950a6251076357579a3d1312e707cf48",
			},
			destination: &destination{
				localIP:          "10.0.0.1",
				sourceIP:         "10.0.0.2",
				wgPort:           wgDefaultPort,
				wgKeepAlive:      wgDefaultKeepalive,
				wgVerboseLogging: false,
			},
		}

		config, err := node.createDestinationWireGuardConfig()
		if err != nil {
			t.Fatalf("createDestinationWireGuardConfig() error = %v, want nil", err)
		}

		expectedIP, _ := netip.ParseAddr("10.0.0.1")
		expectedDNS, _ := parseDNSAddresses([]string{"8.8.8.8", "8.8.4.4"})

		expectedConfig := wireguard.WireGuardConfig{
			PrivateKey:          node.config.privateKey,
			PeerPublicKey:       node.config.peerPublicKey,
			MTU:                 node.config.mtu,
			DNSAddresses:        expectedDNS,
			LocalAddresses:      []netip.Addr{expectedIP},
			ListenPort:          &node.destination.wgPort,
			PersistentKeepalive: &node.destination.wgKeepAlive,
			AllowedIP:           buildAllowedIP(node.destination.sourceIP),
			VerboseLogging:      node.destination.wgVerboseLogging,
		}

		if !reflect.DeepEqual(config, expectedConfig) {
			t.Errorf("createDestinationWireGuardConfig() = %v, want %v", config, expectedConfig)
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
			destination: &destination{
				localIP:          "invalid-ip",
				sourceIP:         "10.0.0.2",
				wgPort:           wgDefaultPort,
				wgKeepAlive:      wgDefaultKeepalive,
				wgVerboseLogging: false,
			},
		}

		_, err := node.createDestinationWireGuardConfig()
		if err == nil {
			t.Fatal("createDestinationWireGuardConfig() expected error for invalid local IP, got nil")
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
			destination: &destination{
				localIP:          "10.0.0.1",
				sourceIP:         "10.0.0.2",
				wgPort:           wgDefaultPort,
				wgKeepAlive:      wgDefaultKeepalive,
				wgVerboseLogging: false,
			},
		}

		_, err := node.createDestinationWireGuardConfig()
		if err == nil {
			t.Fatal("createDestinationWireGuardConfig() expected error for invalid DNS, got nil")
		}
	})
}
