> [!IMPORTANT]
> EdgeLink is no longer maintained and this repository is now archived.

# EdgeLink

EdgeLink is a Go library designed to establish secure, point-to-point network links between two applications. It leverages a userspace WireGuard implementation to create secure tunnels, making it ideal for containerized environments without requiring specific container permissions.

It enables secure bi-directional communication between applications that reside on different networks. A secure link can be established as long as one of the application (the **source**) can reach the other application (the **destination**).

# Requirements

To establish a secure link using EdgeLink, the following conditions must be met:

1. **Network Reachability**:

   - The **source node** must be able to reach the **destination node** over UDP. This is the primary requirement for establishing a connection.

2. **Key Pair Configuration**:

   - Since EdgeLink utilizes WireGuard for secure communication, both nodes need to be configured with cryptographic key pairs.
   - By default, EdgeLink will automatically generate a key pair for each node if none is provided, and it will handle the key exchange process seamlessly.
   - Alternatively, you can specify your own set of key pairs for each node, which will bypass the key exchange process.

3. **TCP Connectivity**:

   - In addition to UDP, the source node must also be able to reach the destination node over TCP on a designated port for the key exchange process.

4. **Default Ports**:

   - The default configuration uses port `51820` for the WireGuard tunnel.
   - Port `50777` is used for the key exchange service.

These requirements ensure that EdgeLink can establish a secure and reliable connection between applications across different networks.

# Quick Start

Install the library:

```bash
go get github.com/portainer/edgelink
```

Create the destination node in your application:

```go
node, err := edgelink.NewNode()
if err != nil {
	log.Fatalf("Failed to create node: %v", err)
}

err = node.SetupAsDestination()
if err != nil {
	log.Fatalf("Failed to setup node as destination node: %v", err)
}

err = node.Link()
if err != nil {
	log.Fatalf("Unable to establish link: %v", err)
}
```

Create the source node in your application:

```go
// This is the public IP address of the destination node
// It must be reachable from the source node
destinationPublicIPAddress := "192.168.1.100"

node, err := edgelink.NewNode()
if err != nil {
	log.Fatalf("Failed to create node: %v", err)
}

err = node.SetupAsSource(destinationPublicIPAddress)
if err != nil {
	log.Fatalf("Failed to setup node as source node: %v", err)
}

err = node.Link()
if err != nil {
	log.Fatalf("Unable to establish link: %v", err)
}
```

The library provides default configuration for both the source and destination nodes. The source node will use the private IP address `10.0.0.2` and the destination node will use the private IP address `10.0.0.1`.

For more information on the configuration options, see the [configuration](#configuration) section below.

Once the link is established, you can use the `GetVirtualNetwork()` method to get the virtual network interface that the two applications are connected to.

```go
import (
	"github.com/portainer/edgelink/examples/ping"
)

// Assuming the node is linked already
// Retrieve the virtual network interface
virtualNetwork := node.GetVirtualNetwork()

// Test the ping between the two nodes
// See the ping implementation in the examples directory for more information
err = ping.TestPingIPv4("10.0.0.2", "10.0.0.1", 1500, 3*time.Second, virtualNetwork)
if err != nil {
	log.Printf("Failed to ping other node: %v", err)
}
```

For more examples, see the [examples](examples) directory.

# Establishing a link

Once the nodes are created and configured, you can establish a link by calling the `Link()` method on both the source and destination nodes.

When called on the destination node and provided that you are using the default configuration, the node will start by listening for incoming key exchange requests on the key exchange port.

Once a key exchange request is received, the node will stop listening on the key exchange port and will create a networking stack and a WireGuard interface that will be used to communicate with the source node.

When called on the source node and provided that you are using the default configuration, the node will start by connecting to the destination node on the key exchange port and will initiate the key exchange process.

Once the key exchange process is complete, the node will create a networking stack and a WireGuard interface that will be used to communicate with the destination node.

# Configuration

## Node configuration

A `Node` is the main object in the library. It holds common configuration for both the source and destination nodes.

The `Node` object is created using the `NewNode` function and uses the Options pattern to allow for configuration.

The `Node` object is then configured as a source or destination node using the `SetupAsSource` or `SetupAsDestination` methods.

### Default configuration

The default configuration will be used if no options are provided. You can find more information about the default values used by the library in the [node.go](node.go) file.

```go
node, err := edgelink.NewNode()
```

### Using custom DNS servers

By default, the library uses 1.1.1.1 as the DNS server. You can use the `WithDNS` option to set one or more custom DNS servers.

```go
node, err := edgelink.NewNode(
	edgelink.WithDNS([]string{"8.8.8.8", "8.8.4.4"}),
)
```

### Enabling verbose logging

By default, the library won't log anything. You can use the `WithLogger` option to set a custom logging function.

```go
func logf(format string, args ...any) {
	format = "edgelink: " + format
	log.Printf(format, args...)
}

node, err := edgelink.NewNode(
	edgelink.WithLogger(logf),
)
```

### Using custom MTU

By default, the library uses an MTU of 1500. You can use the `WithMTU` option to set a custom MTU.

```go
node, err := edgelink.NewNode(
	edgelink.WithMTU(1400),
)
```

### Using a predefined set of keys for WireGuard

By default, the library will generate a key pair for each node if one is not provided and take care of the key exchange process. You can use the `WithPrivateKey` and `WithPeerPublicKey` options to use a predefined set of keys.

> [!NOTE]
> Note that when both of these options are provided, the library will use the provided keys and won't initiate the key exchange process.

These keys can be generated using the `wg` CLI. See https://www.wireguard.com/quickstart/#key-generation for more information.

```go
sourceNode, err := edgelink.NewNode(
	// This is the private key of the source node
	edgelink.WithPrivateKey("WKriFZV6wu0PHonDnpjf9u84oIDLL8FgKB025lAxrnA="),
	// This is the public key of the destination node
	edgelink.WithPeerPublicKey("NAqot6ASJg3QyDQXpcGqsQYAbhp60gTgsGByN0lKnCk="),
)
```

### Persisting the node configuration

By default, the library will keep the node configuration in memory. You can use the `WithPersistentConfig` option to persist the node configuration to a file.

```go
node, err := edgelink.NewNode(
	edgelink.WithPersistentConfig("/path/to/config.yaml"),
)
```

This is particularly useful when running the library in a containerized environment, as it allows the node configuration to be persisted across container restarts.

## Destination node configuration

A node can be configured as a destination node using the `SetupAsDestination` method. It uses the Options pattern to allow for configuration.

### Default configuration

The default configuration will be used if no options are provided.

```go
err := node.SetupAsDestination()
```

### Using a custom key exchange port

By default, the library will use port `50777` for the key exchange service. This is the port that the destination node listens on for incoming key exchange requests from the source node.

> [!NOTE]
> If you have specified a predefined set of keys for the node, the key exchange process will be skipped and setting a custom port will have no effect.

You can use the `WithDestinationKeyxPort` option to set a custom port.

```go
err := node.SetupAsDestination(
	edgelink.WithDestinationKeyxPort(7777),
)
```

### Overriding the private IP addresses associated with the nodes

By default, the library will use the private IP addresses `10.0.0.1` and `10.0.0.2` for the destination and source nodes respectively.

You can use the `WithDestinationLocalIP` and `WithDestinationOriginIP` options to set custom IP addresses.

> [!WARNING]
> Make sure to use the same IP address configuration on the source node using the `SetupAsSource` method.

```go
err := node.SetupAsDestination(
	// This is the local private IP address of the destination node
	edgelink.WithDestinationLocalIP("192.168.1.100"),
	// This is the private IP address of the source node
	edgelink.WithDestinationOriginIP("192.168.1.101"),
)
```

### Using a custom WireGuard port

By default, the library will use port `51820` for the WireGuard tunnel. You can use the `WithDestinationWGPort` option to set a custom port.

```go
err := node.SetupAsDestination(
	edgelink.WithDestinationWGPort(51821),
)
```

### Changing the WireGuard keepalive interval

By default, the library will use a keepalive interval of 25 seconds for the WireGuard tunnel. You can use the `WithDestinationWGKeepalive` option to set a custom keepalive interval.

```go
err := node.SetupAsDestination(
	edgelink.WithDestinationWGKeepalive(10),
)
```

### Enabling verbose logging for the WireGuard library

By default, the underlying WireGuard implementation of the library does not log anything. You can use the `WithDestinationWGVerboseLogging` option to enable verbose logging.

This is useful for debugging the WireGuard implementation.

```go
err := node.SetupAsDestination(
	edgelink.WithDestinationWGVerboseLogging(true),
)
```

## Source node configuration

A node can be configured as a source node using the `SetupAsSource` method. It uses the Options pattern to allow for configuration.

### Default configuration

The default configuration will be used if no options are provided.

```go
err := node.SetupAsSource()
```

### Using a custom key exchange port

By default, the library will use port `50777` for the key exchange service. This is the public port that the source node will connect to on the destination node to initiate the key exchange process.

> [!NOTE]
> You only need to use this option if you have specified a custom key exchange port on the destination node using the `WithDestinationKeyxPort` option.

```go
err := node.SetupAsSource(
	edgelink.WithSourceKeyxPort(7777),
)
```

### Overriding the default key exchange configuration

By default, the library will use a retry interval of 15 seconds and a timeout of 3 seconds for the key exchange process.

You can use the `WithSourceKeyxRetryInterval` and `WithSourceKeyxTimeout` options to set custom values.

```go
err := node.SetupAsSource(
	edgelink.WithSourceKeyxRetryInterval(10 * time.Second),
	edgelink.WithSourceKeyxTimeout(5 * time.Second),
)
```

### Overriding the private IP addresses associated with the nodes

By default, the library will use the private IP addresses `10.0.0.1` and `10.0.0.2` for the destination and source nodes respectively.

You can use the `WithSourceLocalIP` and `WithSourceTargetIP` options to set custom IP addresses.

> [!WARNING]
> Make sure to use the same IP address configuration on the destination node using the `SetupAsDestination` method.

```go
err := node.SetupAsSource(
	// This is the local private IP address of the source node
	edgelink.WithSourceLocalIP("192.168.1.100"),
	// This is the private IP address of the destination node
	edgelink.WithSourceTargetIP("192.168.1.101"),
)
```

### Using a custom WireGuard port

By default, the library will use port `51820` for the WireGuard tunnel. You can use the `WithSourceWGPort` option to set a custom port.

> [!NOTE]
> You only need to use this option if you have specified a custom WireGuard port on the destination node using the `WithDestinationWGPort` option.

```go
err := node.SetupAsSource(
	edgelink.WithSourceWGPort(51821),
)
```

### Enabling verbose logging for the WireGuard library

By default, the underlying WireGuard implementation of the library does not log anything. You can use the `WithSourceWGVerboseLogging` option to enable verbose logging.

This is useful for debugging the WireGuard implementation.

```go
err := node.SetupAsSource(
	edgelink.WithSourceWGVerboseLogging(true),
)
```

# Testing

To run the tests, use the following command:

```bash
go test
```

# License

EdgeLink is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
