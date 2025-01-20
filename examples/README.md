# Examples

This document provides examples of setting up source and destination nodes for various communication methods.

## Simple Ping Test

This example shows how to set up a source and destination node to perform a ping test.

- Source code: [`ping/src/source.go`](ping/src/source.go)
- Destination code: [`ping/dest/destination.go`](ping/dest/destination.go)

## Using Your Own Key Pairs

This example demonstrates setting up nodes with custom key pairs.

### Key Generation

Generate key pairs using the Wireguard CLI:

```bash
wg genkey | tee source.privatekey | wg pubkey > source.publickey
wg genkey | tee destination.privatekey | wg pubkey > destination.publickey
```

### Configuration

- For the source node, use `source.privatekey` and `destination.publickey` with the `-private-key` and `-peer-public-key` flags.
- For the destination node, use `destination.privatekey` and `source.publickey` with the same flags.

- Source code: [`keys/src/source.go`](keys/src/source.go)
- Destination code: [`keys/dest/destination.go`](keys/dest/destination.go)

## Simple TCP Communication

This example illustrates setting up nodes for TCP communication.

- Source code: [`tcp/src/source.go`](tcp/src/source.go)
- Destination code: [`tcp/dest/destination.go`](tcp/dest/destination.go)

## Simple GRPC Communication

This example demonstrates setting up nodes for GRPC communication.

- Source code: [`grpc/src/source.go`](grpc/src/source.go)
- Destination code: [`grpc/dest/destination.go`](grpc/dest/destination.go)
