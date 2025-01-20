// Package edgelink is designed to establish point-to-point connections
// between two nodes, enabling secure communication and data exchange.
// It facilitates the setup and configuration of tunnels between nodes
// that are not in the same network, such as when one node is operating
// at the edge and the other in the cloud. One node initiates the
// establishment of the tunnel, while the other acts as the target.
// The tunnel is powered by WireGuard, ensuring secure and efficient
// connectivity.

package edgelink

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/portainer/edgelink/wireguard"
	"golang.zx2c4.com/wireguard/tun/netstack"
	"gopkg.in/yaml.v3"
)

const (
	// keyxDefaultTimeout is the default timeout duration for key exchange operations.
	keyxDefaultTimeout = 3 * time.Second

	// keyxDefaultPort is the default port used for the key exchange service.
	keyxDefaultPort = 50777

	// keyxDefaultRetryInterval is the default interval between retry attempts for key exchange.
	keyxDefaultRetryInterval = 15 * time.Second

	// wgDefaultPort is the default port used for WireGuard tunnels.
	wgDefaultPort = 51820

	// wgDefaultMTU is the default Maximum Transmission Unit size for WireGuard interfaces.
	wgDefaultMTU = 1500

	// wgDefaultKeepalive is the default interval in seconds for sending
	// keepalive packets in WireGuard.
	wgDefaultKeepalive = 25

	// wgDefaultDestinationLocalIP is the default local IP address for
	// the destination node in WireGuard.
	wgDefaultDestinationLocalIP = "10.0.0.1"

	// wgDefaultSourceLocalIP is the default local IP address for
	// the source node in WireGuard.
	wgDefaultSourceLocalIP = "10.0.0.2"

	// wgDefaultDNS is the default DNS server address used in WireGuard
	// configurations.
	wgDefaultDNS = "1.1.1.1"
)

// Node represents a network node within the edgelink system.
// A node can be either a source node or a destination node.
// The source node is responsible for initiating the connection
// to the destination node.
// The destination node is responsible for accepting the connection
// from the source node.
// It holds configuration details and manages the virtual network
// interface for secure communication.
type Node struct {
	config         nodeConfig
	source         *source
	destination    *destination
	virtualNetwork *netstack.Net
	logf           func(format string, args ...any)
}

// nodeConfig holds the configuration parameters for a Node.
type nodeConfig struct {
	privateKey    string
	publicKey     string
	peerPublicKey string
	mtu           int
	dns           []string
	configPath    string
	logf          func(format string, args ...any)
}

// nodeConfigYAML is used for serializing and deserializing
// node configuration to and from YAML format.
type nodeConfigYAML struct {
	PrivateKey    string   `yaml:"privateKey"`
	PublicKey     string   `yaml:"publicKey"`
	PeerPublicKey string   `yaml:"peerPublicKey"`
	MTU           int      `yaml:"mtu"`
	DNS           []string `yaml:"dns"`
}

// NodeOption is a function type used to modify the nodeConfig
// during the creation of a Node.
type NodeOption func(*nodeConfig)

// WithDNS sets the DNS server addresses for the Node.
func WithDNS(dns []string) NodeOption {
	return func(config *nodeConfig) {
		config.dns = dns
	}
}

// WithLogger sets a custom logging function for the Node.
func WithLogger(logf func(format string, args ...any)) NodeOption {
	return func(config *nodeConfig) {
		config.logf = logf
	}
}

// WithMTU sets the MTU (Maximum Transmission Unit) for the Node.
func WithMTU(mtu int) NodeOption {
	return func(config *nodeConfig) {
		config.mtu = mtu
	}
}

// WithPeerPublicKey sets the peer's public key for the Node, which
// must be base64 encoded. For a source node, this is the public key
// of the destination node. For a destination node, this is the public
// key of the source node.
// This key can be generated via the wireguard CLI. See
// https://www.wireguard.com/quickstart/#key-generation for more
// information.
func WithPeerPublicKey(peerPublicKey string) NodeOption {
	return func(config *nodeConfig) {
		config.peerPublicKey = peerPublicKey
	}
}

// WithPersistentConfig sets the path to a persistent configuration
// file for the Node. If not specified, the node will keep its
// configuration in memory.
func WithPersistentConfig(configPath string) NodeOption {
	return func(config *nodeConfig) {
		config.configPath = configPath
	}
}

// WithPrivateKey sets the private key for the Node, which must be
// base64 encoded. This key can be generated via the wireguard CLI.
// See https://www.wireguard.com/quickstart/#key-generation for more
// information.
func WithPrivateKey(privateKey string) NodeOption {
	return func(config *nodeConfig) {
		config.privateKey = privateKey
	}
}

// NewNode creates a new Node with the specified options. It initializes
// the node configuration, validates it, and sets up the necessary
// cryptographic keys. If a persistent configuration file is specified,
// it attempts to load the configuration from the file. If the file is
// not found, the configuration will be persisted in the specified file
// at the end of the setup process. If a private key is specified in the
// options, it will parse it and derive the public key from it. Otherwise,
// it will generate a new key pair.
//
// Parameters:
// - options: A variadic list of NodeOption functions to customize the Node.
//
// Returns:
// - *Node: A pointer to the newly created Node.
// - error: An error object if the creation fails.
//
// Examples:
//
// Using an existing key pair combination for private key and peer public key:
//
//	node, err := edgelink.NewNode(
//		edgelink.WithPrivateKey("eHU4yujDPbTDaMPve6X6dJUKYlEHY1dXmj0TEucHz0g="),
//		edgelink.WithPeerPublicKey("65s5nHu2KQPkc1Pz51KhHGF0HRHYEZvjJy3X0gT7nVs="),
//	)
//
// Persisting the configuration on disk:
//
//	node, err := edgelink.NewNode(
//		edgelink.WithPersistentConfig("/tmp/node-config.yaml"),
//	)
//
// Setting a custom MTU and DNS:
//
//	node, err := edgelink.NewNode(
//		edgelink.WithMTU(1400),
//		edgelink.WithDNS([]string{"8.8.8.8", "8.8.4.4"}),
//	)
func NewNode(options ...NodeOption) (*Node, error) {
	config := initializeConfig(options)

	if err := validateNodeConfig(config); err != nil {
		return nil, err
	}

	if node, err := loadConfigFromFile(config); err == nil {
		return node, nil
	}

	if err := setupNodeKeyPair(&config); err != nil {
		return nil, err
	}

	if err := handlePeerPublicKey(&config); err != nil {
		return nil, err
	}

	node := &Node{
		config: config,
		logf:   config.logf,
	}

	if err := node.persistConfigChanges(); err != nil {
		return nil, fmt.Errorf("failed to write node configuration changes on disk: %w", err)
	}

	return node, nil
}

// GetVirtualNetwork returns the virtual network interface associated
// with the Node. This interface is used for secure communication
// between nodes.
// Make sure to call Link() before calling this method otherwise it
// will return nil.
//
// Returns:
// - *netstack.Net: A pointer to the virtual network interface.
func (n *Node) GetVirtualNetwork() *netstack.Net {
	return n.virtualNetwork
}

// Link establishes a connection between the Node and its peer.
// It requires the Node to be configured as either a source or
// a destination. The method initiates or accepts a link based
// on the Node's configuration.
//
// Returns:
// - error: An error object if the linking process fails.
func (n *Node) Link() error {
	if n.source == nil && n.destination == nil {
		return fmt.Errorf("node is not configured as source or destination. Call SetupAsSource or SetupAsDestination first")
	}

	if n.source != nil {
		return n.initiateLink()
	}

	return n.acceptLink()
}

func discardLogf(format string, args ...any) {}

func initializeConfig(options []NodeOption) nodeConfig {
	config := nodeConfig{
		mtu:  wgDefaultMTU,
		dns:  []string{wgDefaultDNS},
		logf: discardLogf,
	}
	for _, option := range options {
		option(&config)
	}
	return config
}

func validateNodeConfig(config nodeConfig) error {
	_, err := parseDNSAddresses(config.dns)
	if err != nil {
		return err
	}

	return nil
}

func loadConfigFromFile(config nodeConfig) (*Node, error) {
	if config.configPath != "" {
		if _, err := os.Stat(config.configPath); err == nil {
			config.logf("Loading node configuration from file: %s", config.configPath)
			return buildNodeFromConfigFile(config.configPath)
		}
	}
	return nil, fmt.Errorf("config file not found")
}

func handlePeerPublicKey(config *nodeConfig) error {
	if config.peerPublicKey != "" {
		if err := parsePeerPublicKey(config); err != nil {
			return fmt.Errorf("unable to parse peer public key: %w", err)
		}
	}
	return nil
}

func setupNodeKeyPair(config *nodeConfig) error {
	if config.privateKey == "" {
		return generateNewKeys(config)
	}
	return parseExistingPrivateKey(config)
}

func generateNewKeys(config *nodeConfig) error {
	privateKey, publicKey, err := wireguard.GenerateKeys()
	if err != nil {
		return err
	}

	config.privateKey = hex.EncodeToString(privateKey[:])
	config.publicKey = hex.EncodeToString(publicKey[:])
	return nil
}

func parseExistingPrivateKey(config *nodeConfig) error {
	privateKey, publicKey, err := wireguard.ParsePrivateKey(config.privateKey)
	if err != nil {
		return fmt.Errorf("unable to parse private key: %w", err)
	}

	config.privateKey = hex.EncodeToString(privateKey[:])
	config.publicKey = hex.EncodeToString(publicKey[:])

	return nil
}

func parsePeerPublicKey(config *nodeConfig) error {
	decodedPeerPublicKey, err := base64.StdEncoding.DecodeString(config.peerPublicKey)
	if err != nil {
		return fmt.Errorf("invalid peer public key, make sure it is a valid base64 string: %w", err)
	}
	config.peerPublicKey = hex.EncodeToString(decodedPeerPublicKey)
	return nil
}

func buildNodeFromConfigFile(configPath string) (*Node, error) {
	yamlData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var yamlConfig nodeConfigYAML
	err = yaml.Unmarshal(yamlData, &yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML config: %w", err)
	}

	config := nodeConfig{
		privateKey:    yamlConfig.PrivateKey,
		publicKey:     yamlConfig.PublicKey,
		peerPublicKey: yamlConfig.PeerPublicKey,
		mtu:           yamlConfig.MTU,
		dns:           yamlConfig.DNS,
		configPath:    configPath,
	}

	err = validateNodeConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid node configuration: %w", err)
	}

	node := &Node{
		config: config,
	}

	return node, nil
}

func (n *Node) persistConfigChanges() error {
	if n.config.configPath == "" {
		return nil
	}

	data, err := yaml.Marshal(n.toYAMLConfig())
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	err = os.WriteFile(n.config.configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

func (n *Node) toYAMLConfig() nodeConfigYAML {
	return nodeConfigYAML{
		PrivateKey:    n.config.privateKey,
		PublicKey:     n.config.publicKey,
		PeerPublicKey: n.config.peerPublicKey,
		MTU:           n.config.mtu,
		DNS:           n.config.dns,
	}
}
