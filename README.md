# Submissions Bootstrap Node

This service acts as a dedicated bootstrap node for the PowerLoom decentralized sequencer network. Its primary purpose is to provide a stable, well-known entry point for other libp2p nodes (like snapshotters and validators) to discover and connect to the network.

By connecting to this bootstrap node, new peers obtain a stable dial target and DHT routing assistance to find other participants; gossipsub mesh formation happens directly between snapshotters and validators.

## Features

-   **Stable Entry Point:** Provides a consistent multiaddress for new nodes to join the network.
-   **Peer Discovery:** Helps other nodes discover more peers in the network via libp2p's DHT.
-   **Discovery-only:** Does **not** run gossipsub. Snapshotters and validators carry mesh traffic; the bootstrap node only accepts dial-ins and serves DHT routing.
-   **Lightweight defaults:** Tight connection limits, bounded libp2p memory, no per-connection info logs unless opted in.

## Resource model (why older builds used 2+ GiB / 300% CPU)

A bootstrap node only needs TCP listen + Kademlia DHT. Prior versions also started **full gossipsub** (700ms heartbeats, peer scoring, flood publish) on every inbound peer, enabled **circuit relay** by default, allowed **500–2000** connections, and logged **every** connect/disconnect at info. That behaves like a mesh participant, not a rendezvous point — the `fix/memory-leak` peerstore GC did not change that.

| Setting | Default (new) | Typical old prod `.env` |
|---|---|---|
| `CONN_MANAGER_HIGH_WATER` | `128` | `800` |
| Gossipsub | off | on (unused, no topic join) |
| `ENABLE_RELAY_SERVICE` | `true` (capped) | relay on, unbounded |
| `RELAY_MAX_RESERVATIONS` | `256` | — |
| `LOG_PEER_CONNECTIONS` | `false` | info log per peer |
| `RCMGR_MEMORY_LIMIT_MB` | `512` | unlimited |

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
    PUBLIC_IP=your.vps.public.ip
    # NAT snapshotters without PUBLIC_IP use bootstrap as AutoRelay static relay (default on)
    # ENABLE_RELAY_SERVICE=true
    # RELAY_MAX_RESERVATIONS=256
    # RELAY_MAX_RESERVATIONS_PER_IP=32
    # CONN_MANAGER_HIGH_WATER=256
    # RCMGR_MEMORY_LIMIT_MB=512
    # LOG_PEER_CONNECTIONS=false
    # LIBP2P_LOGGING=warn
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

## Debugging Memory / Performance

### Periodic Status Logs

Every 60 seconds the node logs a status line with key metrics:

```
Status: connected=150 peerstore=152 dht_rt=20 goroutines=45 heap_alloc=28MB heap_inuse=32MB sys=55MB
```

| Metric | What to watch for |
|---|---|
| `peerstore` growing >> `connected` | Peerstore GC not cleaning fast enough |
| `dht_rt` growing unbounded | DHT routing table accumulating entries |
| `goroutines` growing | Goroutine leak |
| `heap_alloc` growing while others stable | Leak in libp2p internals (gossipsub, relay, etc.) |

### pprof Endpoint

Set `PPROF_PORT=6060` in your `.env` file to enable the Go pprof debug server. The port is already wired in `docker-compose.yaml`.

```bash
# Heap profile — what's using memory right now
go tool pprof http://localhost:6060/debug/pprof/heap

# Allocations — what's been allocating the most over time
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap

# Compare two snapshots to find what grew (most useful)
curl -o heap1.pb.gz http://localhost:6060/debug/pprof/heap
# ... wait 30 min ...
curl -o heap2.pb.gz http://localhost:6060/debug/pprof/heap
go tool pprof -base heap1.pb.gz heap2.pb.gz

# Goroutine dump
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

The pprof diff (`-base`) is the most powerful — it shows exactly which allocations grew in the window, narrowing down whether the source is peerstore, DHT, gossipsub, relay, or something else.

## Usage with Other Nodes

To configure other libp2p nodes (like the `snapshotter-lite-local-collector` or `submission-topic-watcher`) to use this bootstrap node, you typically pass its full multiaddress via a command-line flag or environment variable (e.g., `--bootstrap` flag for the watcher, or `BOOTSTRAP_NODE_ADDR` environment variable for the collector).