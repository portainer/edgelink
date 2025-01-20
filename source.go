package edgelink

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	keyx "github.com/portainer/edgelink/keyx/api/v1"
	"github.com/portainer/edgelink/wireguard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// source holds the configuration for a source node in the edgelink system.
// It includes settings for key exchange and WireGuard operations.
type source struct {
	// destinationPublicIP is the public IP address of the destination node.
	// It is the IP address of the destination node that the source node
	// will connect to to initiate WireGuard connections.
	// This must be reachable from the source node.
	destinationPublicIP string

	// keyxPort is the port used for key exchange operations.
	// It is the port on the destination node that the source node will
	// send key exchange requests to.
	keyxPort int

	// keyxRetryInterval is the interval between retry attempts for key exchange.
	keyxRetryInterval time.Duration

	// keyxTimeout is the timeout duration for key exchange operations.
	keyxTimeout time.Duration

	// wgPort is the port used for WireGuard operations.
	// It is the port on the destination node that the source node will
	// connect to to initiate WireGuard connections. It is used in conjunction
	// with the destinationPublicIP to build the WireGuard endpoint.
	wgPort int

	// wgVerboseLogging enables verbose logging for the underlying WireGuard
	// library.
	wgVerboseLogging bool

	// localIP is the local IP address assigned to the source node.
	localIP string

	// destinationIP is the IP address of the destination node.
	// It is used to build the allowed IP for the source node.
	destinationIP string
}

// SourceOption is a function type used to modify the source configuration during its setup.
type SourceOption func(*source)

// WithSourceKeyxPort sets the key exchange port for the source.
func WithSourceKeyxPort(port int) SourceOption {
	return func(src *source) {
		src.keyxPort = port
	}
}

// WithSourceKeyxRetryInterval sets the retry interval for key exchange operations at the source.
func WithSourceKeyxRetryInterval(interval time.Duration) SourceOption {
	return func(src *source) {
		src.keyxRetryInterval = interval
	}
}

// WithSourceKeyxTimeout sets the timeout duration for key exchange operations at the source.
func WithSourceKeyxTimeout(timeout time.Duration) SourceOption {
	return func(src *source) {
		src.keyxTimeout = timeout
	}
}

// WithSourceLocalIP sets the local IP address for the source node.
func WithSourceLocalIP(ip string) SourceOption {
	return func(src *source) {
		src.localIP = ip
	}
}

// WithSourceTargetIP sets the IP address of the destination node for the source.
// It is used to build the allowed IP for the source node.
func WithSourceTargetIP(ip string) SourceOption {
	return func(src *source) {
		src.destinationIP = ip
	}
}

// WithSourceWGPort sets the WireGuard port for the source.
func WithSourceWGPort(port int) SourceOption {
	return func(src *source) {
		src.wgPort = port
	}
}

// WithSourceWGVerboseLogging enables or disables verbose logging for the
// underlying WireGuard library.
func WithSourceWGVerboseLogging(verbose bool) SourceOption {
	return func(src *source) {
		src.wgVerboseLogging = verbose
	}
}

// SetupAsSource configures a Node as a source node using the provided options.
// It initializes and validates the source configuration.
// A source node will setup a key exchange client if no peer public key is
// provided and will keep retrying until the key exchange succeeds.
//
// Parameters:
// - destinationPublicIP: The public IP address of the destination node.
// - options: A variadic list of SourceOption functions to customize the source.
//
// Returns:
// - error: An error object if the setup fails.
func (n *Node) SetupAsSource(destinationPublicIP string, options ...SourceOption) error {
	s := &source{
		destinationPublicIP: destinationPublicIP,
		keyxPort:            keyxDefaultPort,
		keyxRetryInterval:   keyxDefaultRetryInterval,
		keyxTimeout:         keyxDefaultTimeout,
		wgPort:              wgDefaultPort,
		wgVerboseLogging:    false,
		localIP:             wgDefaultSourceLocalIP,
		destinationIP:       wgDefaultDestinationLocalIP,
	}

	for _, option := range options {
		option(s)
	}

	err := validateSource(s)
	if err != nil {
		return fmt.Errorf("invalid source configuration: %w", err)
	}

	n.source = s

	return nil
}

func validateSource(s *source) error {
	if s.destinationPublicIP == "" {
		return fmt.Errorf("DestinationPublicIP is required")
	}

	_, err := netip.ParseAddr(s.localIP)
	if err != nil {
		return fmt.Errorf("invalid local IP address: %w", err)
	}

	_, err = netip.ParseAddr(s.destinationIP)
	if err != nil {
		return fmt.Errorf("invalid destination IP address: %w", err)
	}

	return nil
}

func (n *Node) initiateLink() error {
	if n.config.peerPublicKey == "" {
		for {
			err := n.tryKeyExchange()
			if err != nil {
				n.logf("warning - %v. Will retry in %v", err, n.source.keyxRetryInterval)
				time.Sleep(n.source.keyxRetryInterval)
				continue
			}

			break
		}
	}

	err := n.persistConfigChanges()
	if err != nil {
		return fmt.Errorf("failed to write node configuration changes on disk: %w", err)
	}

	err = n.createSourceVirtualNetwork()
	if err != nil {
		return fmt.Errorf("source node failed to create virtual network: %w", err)
	}

	return nil
}

func (n *Node) tryKeyExchange() error {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	keyxAddress := fmt.Sprintf("%s:%d", n.source.destinationPublicIP, n.source.keyxPort)

	conn, err := grpc.NewClient(keyxAddress, opts...)
	if err != nil {
		return fmt.Errorf("source node failed to dial destination node for key exchange: %w", err)
	}
	defer conn.Close()

	client := keyx.NewKeyExchangeClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), n.source.keyxTimeout)
	defer cancel()

	r, err := client.ExchangeKeys(ctx, &keyx.PublicKey{Key: n.config.publicKey})
	if err != nil {
		return fmt.Errorf("source node failed to exchange key with destination node: %w", err)
	}

	n.config.peerPublicKey = r.Key
	n.logf("Received public key from server: %s", r.Key)

	return nil
}

func (n *Node) createSourceWireGuardConfig() (wireguard.WireGuardConfig, error) {
	endpoint := fmt.Sprintf("%s:%d", n.source.destinationPublicIP, n.source.wgPort)

	ip, err := netip.ParseAddr(n.source.localIP)
	if err != nil {
		return wireguard.WireGuardConfig{}, fmt.Errorf("invalid local IP address: %w", err)
	}

	nameservers, err := parseDNSAddresses(n.config.dns)
	if err != nil {
		return wireguard.WireGuardConfig{}, fmt.Errorf("invalid DNS address: %w", err)
	}

	return wireguard.WireGuardConfig{
		PrivateKey:     n.config.privateKey,
		PeerPublicKey:  n.config.peerPublicKey,
		MTU:            n.config.mtu,
		DNSAddresses:   nameservers,
		LocalAddresses: []netip.Addr{ip},
		AllowedIP:      buildAllowedIP(n.source.destinationIP),
		Endpoint:       &endpoint,
		VerboseLogging: n.source.wgVerboseLogging,
	}, nil
}

func (n *Node) createSourceVirtualNetwork() error {
	wgConfig, err := n.createSourceWireGuardConfig()
	if err != nil {
		return fmt.Errorf("source node failed to create wireguard config: %w", err)
	}

	vNet, err := wireguard.CreateVirtualNetwork(wgConfig)
	if err != nil {
		return fmt.Errorf("source node failed to create virtual network: %w", err)
	}

	n.virtualNetwork = vNet
	return nil
}
