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

## Docker with Docker Compose

To build and run the bootstrap node using Docker Compose, follow these steps:

1.  **Build the Docker Image:**

    ```bash
    docker-compose build
    ```

2.  **Start the Docker Container:**

    ```bash
    ./start.sh
    ```

3.  **Stop the Docker Container:**

    ```bash
    ./stop.sh
    ```

    You can view the logs of the running service:

    ```bash
    docker-compose logs -f bootstrap-node
    ```

## Configuration

To ensure a consistent Peer ID and multiaddress for your bootstrap node, you should configure it with a static private key. If no private key is provided, a new one will be generated on each startup, resulting in a different Peer ID and multiaddress.

1.  **Generate a Private Key:**

    You can generate a new private key and its corresponding Peer ID and multiaddress by running the bootstrap node executable with the `--generate-key` flag:

    ```bash
    go run ./cmd/main.go --generate-key
    ```

    This will output the generated private key (hex-encoded, 128 characters long), the derived Peer ID, and a local placeholder multiaddress. Copy the `Generated Private Key (hex)` value.

2.  **Configure the `.env` file:**

    Create a `.env` file in the same directory as `docker-compose.yaml` (if it doesn't exist):

    ```bash
    cp .env.example .env
    ```

    Edit the `.env` file and set the `PRIVATE_KEY` variable with the hex-encoded private key generated in the previous step.

    ```dotenv
    PRIVATE_KEY=your_generated_private_key_here
    ```

## Run (Local Executable)

You can run the bootstrap node locally (without Docker) on a default port or specify a custom one.

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
