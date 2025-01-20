package edgelink

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestNewNode(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		expectedConfig := nodeConfig{
			dns:           []string{wgDefaultDNS},
			mtu:           wgDefaultMTU,
			configPath:    "",
			peerPublicKey: "",
		}

		node, err := NewNode()
		if err != nil {
			t.Fatalf("NewNode() error = %v, want nil", err)
		}

		if !reflect.DeepEqual(node.config.dns, expectedConfig.dns) {
			t.Errorf("DNS = %v, want %v", node.config.dns, expectedConfig.dns)
		}

		if node.config.mtu != expectedConfig.mtu {
			t.Errorf("MTU = %v, want %v", node.config.mtu, expectedConfig.mtu)
		}

		if node.config.configPath != expectedConfig.configPath {
			t.Errorf("ConfigPath = %v, want %v", node.config.configPath, expectedConfig.configPath)
		}

		if node.config.privateKey == "" {
			t.Errorf("PrivateKey is not set")
		}

		if _, err := hex.DecodeString(node.config.privateKey); err != nil {
			t.Errorf("PrivateKey is not a valid hex string: %v", err)
		}

		if node.config.publicKey == "" {
			t.Errorf("PublicKey is not set")
		}

		if _, err := hex.DecodeString(node.config.publicKey); err != nil {
			t.Errorf("PublicKey is not a valid hex string: %v", err)
		}

		if node.config.peerPublicKey != expectedConfig.peerPublicKey {
			t.Errorf("PeerPublicKey = %v, want %v", node.config.peerPublicKey, expectedConfig.peerPublicKey)
		}
	})

	t.Run("PrivateAndPeerPublicKeySet", func(t *testing.T) {
		options := []NodeOption{
			WithDNS([]string{"8.8.8.8", "8.8.4.4"}),
			WithMTU(1500),
			WithPeerPublicKey("65s5nHu2KQPkc1Pz51KhHGF0HRHYEZvjJy3X0gT7nVs="),
			WithPrivateKey("eHU4yujDPbTDaMPve6X6dJUKYlEHY1dXmj0TEucHz0g="),
		}
		expectedConfig := nodeConfig{
			dns:           []string{"8.8.8.8", "8.8.4.4"},
			mtu:           1500,
			peerPublicKey: "eb9b399c7bb62903e47353f3e752a11c61741d11d8119be3272dd7d204fb9d5b",
			privateKey:    "787538cae8c33db4c368c3ef7ba5fa74950a6251076357579a3d1312e707cf48",
			publicKey:     "340aa8b7a012260dd0c83417a5c1aab106006e1a7ad204e0b0607237494a9c29",
		}

		node, err := NewNode(options...)
		if err != nil {
			t.Fatalf("NewNode() error = %v, want nil", err)
		}

		if !reflect.DeepEqual(node.config.dns, expectedConfig.dns) {
			t.Errorf("DNS = %v, want %v", node.config.dns, expectedConfig.dns)
		}

		if node.config.mtu != expectedConfig.mtu {
			t.Errorf("MTU = %v, want %v", node.config.mtu, expectedConfig.mtu)
		}

		if node.config.peerPublicKey != expectedConfig.peerPublicKey {
			t.Errorf("PeerPublicKey = %v, want %v", node.config.peerPublicKey, expectedConfig.peerPublicKey)
		}

		if node.config.privateKey != expectedConfig.privateKey {
			t.Errorf("PrivateKey = %v, want %v", node.config.privateKey, expectedConfig.privateKey)
		}

		if node.config.publicKey != expectedConfig.publicKey {
			t.Errorf("PublicKey = %v, want %v", node.config.publicKey, expectedConfig.publicKey)
		}
	})

	t.Run("InvalidPeerPublicKey", func(t *testing.T) {
		options := []NodeOption{
			WithPeerPublicKey("invalid_base64_key"),
		}

		_, err := NewNode(options...)
		if err == nil {
			t.Error("NewNode() expected error for invalid peer public key, got nil")
		}
	})

	t.Run("InvalidPrivateKey", func(t *testing.T) {
		options := []NodeOption{
			WithPrivateKey("invalid_base64_key"),
		}

		_, err := NewNode(options...)
		if err == nil {
			t.Error("NewNode() expected error for invalid private key, got nil")
		}
	})

	t.Run("InvalidDNSAddresses", func(t *testing.T) {
		options := []NodeOption{
			WithDNS([]string{"invalid_dns"}),
		}

		_, err := NewNode(options...)
		if err == nil {
			t.Error("NewNode() expected error for invalid DNS addresses, got nil")
		}
	})
}

func TestNodeOptions(t *testing.T) {
	dns := []string{"8.8.8.8", "8.8.4.4"}
	var logCalled bool
	logFunc := func(format string, args ...any) {
		logCalled = true
	}
	mtu := 1500
	peerPublicKey := "base64EncodedPublicKey"
	configPath := "/path/to/config"
	privateKey := "base64EncodedPrivateKey"

	options := []NodeOption{
		WithDNS(dns),
		WithLogger(logFunc),
		WithMTU(mtu),
		WithPeerPublicKey(peerPublicKey),
		WithPersistentConfig(configPath),
		WithPrivateKey(privateKey),
	}

	config := initializeConfig(options)

	if !reflect.DeepEqual(config.dns, dns) {
		t.Errorf("DNS = %v, want %v", config.dns, dns)
	}

	config.logf("test")
	if !logCalled {
		t.Error("Logger function was not set correctly")
	}

	if config.mtu != mtu {
		t.Errorf("MTU = %v, want %v", config.mtu, mtu)
	}

	if config.peerPublicKey != peerPublicKey {
		t.Errorf("PeerPublicKey = %v, want %v", config.peerPublicKey, peerPublicKey)
	}

	if config.configPath != configPath {
		t.Errorf("ConfigPath = %v, want %v", config.configPath, configPath)
	}

	if config.privateKey != privateKey {
		t.Errorf("PrivateKey = %v, want %v", config.privateKey, privateKey)
	}
}

func TestNodeLink_NotConfigured(t *testing.T) {
	node := &Node{}

	err := node.Link()
	if err == nil {
		t.Fatal("Link() expected error, got nil")
	}

	expectedError := "node is not configured as source or destination. Call SetupAsSource or SetupAsDestination first"
	if err.Error() != expectedError {
		t.Errorf("Link() error = %v, want %v", err.Error(), expectedError)
	}
}

func TestValidateNodeConfig(t *testing.T) {
	t.Run("ValidDNSAddresses", func(t *testing.T) {
		config := nodeConfig{
			dns: []string{"8.8.8.8", "8.8.4.4"},
		}

		err := validateNodeConfig(config)
		if err != nil {
			t.Errorf("validateNodeConfig() error = %v, want nil", err)
		}
	})

	t.Run("InvalidDNSAddresses", func(t *testing.T) {
		config := nodeConfig{
			dns: []string{"invalid-ip"},
		}

		err := validateNodeConfig(config)
		if err == nil {
			t.Fatal("validateNodeConfig() expected error, got nil")
		}
	})
}

func TestSetupNodeKeyPair(t *testing.T) {
	t.Run("PrivateKeyNotSet", func(t *testing.T) {
		config := &nodeConfig{privateKey: ""}

		err := setupNodeKeyPair(config)
		if err != nil {
			t.Errorf("setupNodeKeyPair() error = %v, want nil", err)
		}

		if config.privateKey == "" {
			t.Errorf("setupNodeKeyPair() privateKey is not set")
		}

		if config.publicKey == "" {
			t.Errorf("setupNodeKeyPair() publicKey is not set")
		}

		if _, err := hex.DecodeString(config.privateKey); err != nil {
			t.Errorf("setupNodeKeyPair() privateKey is not a valid hex string: %v", err)
		}

		if _, err := hex.DecodeString(config.publicKey); err != nil {
			t.Errorf("setupNodeKeyPair() publicKey is not a valid hex string: %v", err)
		}
	})

	t.Run("PrivateKeySet", func(t *testing.T) {
		config := &nodeConfig{privateKey: "eHU4yujDPbTDaMPve6X6dJUKYlEHY1dXmj0TEucHz0g="}

		err := setupNodeKeyPair(config)
		if err != nil {
			t.Errorf("setupNodeKeyPair() error = %v, want nil", err)
		}

		expectedHexPrivateKey := "787538cae8c33db4c368c3ef7ba5fa74950a6251076357579a3d1312e707cf48"
		if config.privateKey != expectedHexPrivateKey {
			t.Errorf("setupNodeKeyPair() privateKey does not match expected value: got %s, want %s", config.privateKey, expectedHexPrivateKey)
		}

		expectedHexPublicKey := "340aa8b7a012260dd0c83417a5c1aab106006e1a7ad204e0b0607237494a9c29"
		if config.publicKey != expectedHexPublicKey {
			t.Errorf("setupNodeKeyPair() publicKey does not match expected value: got %s, want %s", config.publicKey, expectedHexPublicKey)
		}
	})

	t.Run("InvalidPrivateKey", func(t *testing.T) {
		config := &nodeConfig{privateKey: "invalid-base64-key"}

		err := setupNodeKeyPair(config)
		if err == nil {
			t.Error("setupNodeKeyPair() expected error, got nil")
		}
	})
}

func TestHandlePeerPublicKey(t *testing.T) {
	t.Run("PeerPublicKeyNotSet", func(t *testing.T) {
		config := &nodeConfig{}

		err := handlePeerPublicKey(config)
		if err != nil {
			t.Errorf("handlePeerPublicKey() error = %v, want nil", err)
		}
	})

	t.Run("PeerPublicKeySet", func(t *testing.T) {
		config := &nodeConfig{peerPublicKey: "65s5nHu2KQPkc1Pz51KhHGF0HRHYEZvjJy3X0gT7nVs="}

		err := handlePeerPublicKey(config)
		if err != nil {
			t.Errorf("handlePeerPublicKey() error = %v, want nil", err)
		}

		// This is the expected hex encoded version of the peer public key.
		expectedHexKey := "eb9b399c7bb62903e47353f3e752a11c61741d11d8119be3272dd7d204fb9d5b"
		if config.peerPublicKey != expectedHexKey {
			t.Errorf("handlePeerPublicKey() peerPublicKey does not match expected value: got %s, want %s", config.peerPublicKey, expectedHexKey)
		}
	})

	t.Run("InvalidPeerPublicKey", func(t *testing.T) {
		config := &nodeConfig{peerPublicKey: "invalid-base64-key"}

		err := handlePeerPublicKey(config)
		if err == nil {
			t.Error("handlePeerPublicKey() expected error, got nil")
		}
	})
}

func TestNodeToYAMLConfig(t *testing.T) {
	node := &Node{
		config: nodeConfig{
			privateKey:    "privateKeyValue",
			publicKey:     "publicKeyValue",
			peerPublicKey: "peerPublicKeyValue",
			mtu:           1500,
			dns:           []string{"8.8.8.8", "8.8.4.4"},
		},
	}

	expectedYAMLConfig := nodeConfigYAML{
		PrivateKey:    "privateKeyValue",
		PublicKey:     "publicKeyValue",
		PeerPublicKey: "peerPublicKeyValue",
		MTU:           1500,
		DNS:           []string{"8.8.8.8", "8.8.4.4"},
	}

	yamlConfig := node.toYAMLConfig()

	if !reflect.DeepEqual(yamlConfig, expectedYAMLConfig) {
		t.Errorf("toYAMLConfig() = %v, want %v", yamlConfig, expectedYAMLConfig)
	}
}
