# Submissions Bootstrap Node

This service acts as a dedicated bootstrap node for the PowerLoom decentralized sequencer network. Its primary purpose is to provide a stable, well-known entry point for other libp2p nodes (like snapshotters and validators) to discover and connect to the network.

By connecting to this bootstrap node, new peers can quickly find other participants in the network, facilitating efficient peer discovery and message propagation for Gossipsub topics.

## Features

-   **Stable Entry Point:** Provides a consistent multiaddress for new nodes to join the network.
-   **Peer Discovery:** Helps other nodes discover more peers in the network via libp2p's DHT.
-   **Lightweight:** Designed to be a simple, robust, and long-running service with minimal overhead.

## Build

To build the `submissions-bootstrap-node` executable, navigate to the service's root directory and run:

```bash
go build ./cmd/main.go
```

This will create an executable named `main` (or `main.exe` on Windows) in the current directory.

## Run

You can run the bootstrap node on a default port or specify a custom one.

### Default Port (4001)

```bash
./main
```

### Custom Port

To run on a specific port (e.g., 4002):

```bash
./main --port=4002
```

### Example Startup Output

When the node starts, it will log its Peer ID and listening multiaddresses. You will need one of these multiaddresses to configure other nodes that wish to connect to this bootstrap node.

```
INFO[2025-07-10T17:16:04+05:30] Libp2p host created with ID: 12D3KooWCNsSau1o9MeMVpHudvHaZRLESRcaGVK9FPKhdLU36BtF
INFO[2025-07-10T17:16:04+05:30] Bootstrap node started. ID: 12D3KooWCNsSau1o9MeMVpHudvHaZRLESRcaGVK9FPKhdLU36BtF
INFO[2025-07-10T17:16:04+05:30] Listening on addresses: [/ip4/127.0.0.1/tcp/4001 /ip4/192.168.0.126/tcp/4001]
```

From the example above, a full multiaddress to use for other nodes would be:
`/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWCNsSau1o9MeMVpHudvHaZRLESRcaGVK9FPKhdLU36BtF`

## Usage with Other Nodes

To configure other libp2p nodes (like the `snapshotter-lite-local-collector` or `submission-topic-watcher`) to use this bootstrap node, you typically pass its full multiaddress via a command-line flag or environment variable (e.g., `--bootstrap` flag for the watcher, or `BOOTSTRAP_NODE_ADDR` environment variable for the collector).
