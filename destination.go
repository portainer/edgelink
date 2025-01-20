package edgelink

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	keyx "github.com/portainer/edgelink/keyx/api/v1"
	"github.com/portainer/edgelink/wireguard"
	"google.golang.org/grpc"
)

// destination holds the configuration for a destination node in the
// edgelink system. It includes settings for key exchange and WireGuard
// operations.
type destination struct {
	// keyxPort is the port used for key exchange operations.
	// It is the port that the destination node listens on for incoming
	// key exchange requests from the source node.
	keyxPort int

	// wgPort is the port used for WireGuard operations.
	// It is the port that the destination node listens on for incoming
	// WireGuard connections from the source node.
	wgPort int

	// wgKeepAlive is the interval for sending keepalive packets in WireGuard.
	wgKeepAlive int

	// wgVerboseLogging enables verbose logging for the underlying WireGuard
	// library.
	wgVerboseLogging bool

	// localIP is the local IP address assigned to the destination node.
	localIP string

	// sourceIP is the IP address of the source node. It is used to build
	// the allowed IP for the destination node.
	sourceIP string
}

// DestinationOption is a function type used to modify the destination
// configuration during its setup.
type DestinationOption func(*destination)

// WithDestinationKeyxPort sets the key exchange port for the destination.
func WithDestinationKeyxPort(port int) DestinationOption {
	return func(dest *destination) {
		dest.keyxPort = port
	}
}

// WithDestinationLocalIP sets the local IP address for the destination node.
func WithDestinationLocalIP(ip string) DestinationOption {
	return func(dest *destination) {
		dest.localIP = ip
	}
}

// WithDestinationOriginIP sets the IP address of the source node. It is used
// to build the allowed IP for the destination node.
func WithDestinationOriginIP(ip string) DestinationOption {
	return func(dest *destination) {
		dest.sourceIP = ip
	}
}

// WithDestinationWGKeepalive sets the keepalive interval for WireGuard
// operations. The destination node will send keepalive packets to the source
// node to maintain the connection using this specified interval.
func WithDestinationWGKeepalive(keepalive int) DestinationOption {
	return func(dest *destination) {
		dest.wgKeepAlive = keepalive
	}
}

// WithDestinationWGPort sets the WireGuard port for the destination.
func WithDestinationWGPort(port int) DestinationOption {
	return func(dest *destination) {
		dest.wgPort = port
	}
}

// WithDestinationWGVerboseLogging enables or disables verbose logging
// for the underlying WireGuard library.
func WithDestinationWGVerboseLogging(verbose bool) DestinationOption {
	return func(dest *destination) {
		dest.wgVerboseLogging = verbose
	}
}

// SetupAsDestination configures a Node as a destination node using the
// provided options. It initializes and validates the destination configuration.
// A destination node will setup a key exchange server if no peer public key is
// provided.
//
// Parameters:
// - options: A variadic list of DestinationOption functions to customize the destination.
//
// Returns:
// - error: An error object if the setup fails.
func (n *Node) SetupAsDestination(options ...DestinationOption) error {
	d := &destination{
		keyxPort:         keyxDefaultPort,
		wgPort:           wgDefaultPort,
		wgKeepAlive:      wgDefaultKeepalive,
		wgVerboseLogging: false,
		localIP:          wgDefaultDestinationLocalIP,
		sourceIP:         wgDefaultSourceLocalIP,
	}

	for _, option := range options {
		option(d)
	}

	err := validateDestination(d)
	if err != nil {
		return fmt.Errorf("invalid destination configuration: %w", err)
	}

	n.destination = d

	return nil
}

func validateDestination(d *destination) error {
	_, err := netip.ParseAddr(d.localIP)
	if err != nil {
		return fmt.Errorf("invalid local IP address: %w", err)
	}

	_, err = netip.ParseAddr(d.sourceIP)
	if err != nil {
		return fmt.Errorf("invalid source IP address: %w", err)
	}

	return nil
}

func (n *Node) acceptLink() error {
	if n.config.peerPublicKey == "" {
		if err := n.startKeyExchangeServer(); err != nil {
			return err
		}
	}

	if err := n.createDestinationVirtualNetwork(); err != nil {
		return fmt.Errorf("destination node failed to create virtual network: %w", err)
	}

	return nil
}

func (n *Node) startKeyExchangeServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", n.destination.keyxPort))
	if err != nil {
		return fmt.Errorf("unable to listen on port %d: %w", n.destination.keyxPort, err)
	}
	defer lis.Close()

	keyExchange := make(chan string)
	defer close(keyExchange)

	grpcServer := grpc.NewServer()
	srv := &keyxServer{publicKey: n.config.publicKey, keyExchange: keyExchange}
	keyx.RegisterKeyExchangeServer(grpcServer, srv)

	errChan := make(chan error, 1)

	go func() {
		errChan <- grpcServer.Serve(lis)
		close(errChan)
	}()

	select {
	case n.config.peerPublicKey = <-srv.keyExchange:
		n.logf("Received public key from client: %s", n.config.peerPublicKey)
		err = n.persistConfigChanges()
		if err != nil {
			return fmt.Errorf("failed to write node configuration changes on disk: %w", err)
		}
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("failed to serve gRPC server: %w", err)
		}
	}

	grpcServer.GracefulStop()
	return nil
}

func (n *Node) createDestinationWireGuardConfig() (wireguard.WireGuardConfig, error) {
	ip, err := netip.ParseAddr(n.destination.localIP)
	if err != nil {
		return wireguard.WireGuardConfig{}, fmt.Errorf("invalid local IP address: %w", err)
	}

	nameservers, err := parseDNSAddresses(n.config.dns)
	if err != nil {
		return wireguard.WireGuardConfig{}, fmt.Errorf("invalid DNS address: %w", err)
	}

	return wireguard.WireGuardConfig{
		PrivateKey:          n.config.privateKey,
		PeerPublicKey:       n.config.peerPublicKey,
		MTU:                 n.config.mtu,
		DNSAddresses:        nameservers,
		LocalAddresses:      []netip.Addr{ip},
		ListenPort:          &n.destination.wgPort,
		PersistentKeepalive: &n.destination.wgKeepAlive,
		AllowedIP:           buildAllowedIP(n.destination.sourceIP),
		VerboseLogging:      n.destination.wgVerboseLogging,
	}, nil
}

func (n *Node) createDestinationVirtualNetwork() error {
	wgConfig, err := n.createDestinationWireGuardConfig()
	if err != nil {
		return fmt.Errorf("failed to create wireguard config: %v", err)
	}

	vNet, err := wireguard.CreateVirtualNetwork(wgConfig)
	if err != nil {
		return fmt.Errorf("failed to create virtual network: %v", err)
	}

	n.virtualNetwork = vNet
	return nil
}

type keyxServer struct {
	keyx.UnimplementedKeyExchangeServer
	publicKey   string
	keyExchange chan string
}

func (s *keyxServer) ExchangeKeys(ctx context.Context, req *keyx.PublicKey) (*keyx.PublicKey, error) {
	s.keyExchange <- req.Key
	return &keyx.PublicKey{Key: s.publicKey}, nil
}
