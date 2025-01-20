// Package wireguard provides functionality to create and configure
// a virtual network interface using WireGuard.

package wireguard

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// WireGuardConfig holds the configuration parameters for setting up
// a WireGuard virtual network interface.
//
// Fields:
// - LocalAddresses: A list of local IP addresses assigned to the interface.
// - DNSAddresses: A list of DNS server addresses for the interface.
// - MTU: The Maximum Transmission Unit size for the interface.
// - PrivateKey: The private key for the WireGuard interface, hex encoded.
// - PeerPublicKey: The public key of the peer to connect to.
// - AllowedIP: The IP range that is allowed to communicate through the interface.
// - ListenPort: The port on which the interface listens for incoming connections.
// - Endpoint: The endpoint address of the peer.
// - PersistentKeepalive: The interval for sending keepalive packets to maintain the connection.
// - VerboseLogging: Enables verbose logging for debugging purposes.
type WireGuardConfig struct {
	// LocalAddresses is a list of local IP addresses assigned to the interface.
	LocalAddresses []netip.Addr

	// DNSAddresses is a list of DNS server addresses for the interface.
	DNSAddresses []netip.Addr

	// MTU is the Maximum Transmission Unit size for the interface.
	MTU int

	// PrivateKey is the private key for the WireGuard interface, which must be hex encoded.
	PrivateKey string

	// PeerPublicKey is the public key of the peer to connect to, which must be hex encoded.
	PeerPublicKey string

	// AllowedIP is the IP range or direct address that is allowed to communicate with the peer.
	// It can be specified as a CIDR notation (e.g., 10.0.0.0/32 or 10.0.0.1/30).
	AllowedIP string

	// ListenPort is an optional port on which the interface listens for incoming connections.
	// It is used to set up the target side of a WireGuard tunnel.
	ListenPort *int

	// Endpoint is an optional address of the peer, used by the origin side to send traffic
	// to the target node's public IP.
	Endpoint *string

	// PersistentKeepalive is an optional interval for sending keepalive packets to maintain
	// the connection, used by the target side.
	PersistentKeepalive *int

	// VerboseLogging enables verbose logging for debugging purposes.
	VerboseLogging bool
}

// CreateVirtualNetwork initializes a virtual network interface using
// the provided WireGuard configuration. It returns a netstack.Net
// object representing the virtual network and an error if the operation
// fails.
//
// Parameters:
// - config: A WireGuardConfig object containing the necessary configuration.
//
// Returns:
// - *netstack.Net: A pointer to the created virtual network.
// - error: An error object if the creation fails.
func CreateVirtualNetwork(config WireGuardConfig) (*netstack.Net, error) {
	tun, vNet, err := netstack.CreateNetTUN(
		config.LocalAddresses,
		config.DNSAddresses,
		config.MTU,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device: %w", err)
	}

	err = configureDevice(tun, config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure WireGuard device: %w", err)
	}

	return vNet, nil
}

// GenerateKeys generates a new WireGuard key pair. It creates a private key
// and derives the corresponding public key from it. This function is useful
// for setting up new WireGuard interfaces where a fresh key pair is needed.
//
// Returns:
// - wgtypes.Key: The generated private key.
// - wgtypes.Key: The derived public key.
// - error: An error object if the key generation fails.
func GenerateKeys() (wgtypes.Key, wgtypes.Key, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return wgtypes.Key{}, wgtypes.Key{}, fmt.Errorf("failed to generate private key: %w", err)
	}
	return privateKey, privateKey.PublicKey(), nil
}

// ParsePrivateKey decodes a base64-encoded private key and derives the
// corresponding public key. This function is useful when you have a
// pre-existing private key in base64 format (such as one generated via
// the wireguard CLI) and need to use it for WireGuard configuration.
//
// Parameters:
// - encodedPrivateKey: A string containing the base64-encoded private key.
//
// Returns:
// - wgtypes.Key: The decoded private key.
// - wgtypes.Key: The derived public key.
// - error: An error object if the decoding or parsing fails.
func ParsePrivateKey(encodedPrivateKey string) (wgtypes.Key, wgtypes.Key, error) {
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(encodedPrivateKey)
	if err != nil {
		return wgtypes.Key{}, wgtypes.Key{}, fmt.Errorf("invalid private key, make sure it is a valid base64 string: %w", err)
	}

	privateKey, err := wgtypes.NewKey(decodedPrivateKey)
	if err != nil {
		return wgtypes.Key{}, wgtypes.Key{}, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, privateKey.PublicKey(), nil
}

// configureDevice uses the cross-platform userspace implementation configuration protocol
// to configure the WireGuard device.
// More info: https://www.wireguard.com/xplatform/
func configureDevice(tun tun.Device, config WireGuardConfig) error {
	var configBuilder strings.Builder

	configBuilder.WriteString(fmt.Sprintf("private_key=%s\n", config.PrivateKey))

	if config.ListenPort != nil {
		configBuilder.WriteString(fmt.Sprintf("listen_port=%d\n", *config.ListenPort))
	}

	configBuilder.WriteString(fmt.Sprintf("public_key=%s\n", config.PeerPublicKey))
	configBuilder.WriteString(fmt.Sprintf("allowed_ip=%s\n", config.AllowedIP))

	if config.Endpoint != nil {
		configBuilder.WriteString(fmt.Sprintf("endpoint=%s\n", *config.Endpoint))
	}

	if config.PersistentKeepalive != nil {
		configBuilder.WriteString(fmt.Sprintf("persistent_keepalive_interval=%d\n", *config.PersistentKeepalive))
	}

	logger := device.NewLogger(device.LogLevelSilent, "")
	if config.VerboseLogging {
		logger = device.NewLogger(device.LogLevelVerbose, "wireguard ")
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), logger)

	err := dev.IpcSet(configBuilder.String())
	if err != nil {
		return fmt.Errorf("failed to set WireGuard configuration: %w", err)
	}

	err = dev.Up()
	if err != nil {
		return fmt.Errorf("failed to bring up WireGuard vNode interface: %w", err)
	}

	return nil
}
