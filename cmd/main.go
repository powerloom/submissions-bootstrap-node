package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"submissions-bootstrap-node/pkg/config"
	"submissions-bootstrap-node/pkg/service"
	"syscall"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Command-line flags
	defaultPort := 4001
	if portStr := os.Getenv("BOOTSTRAP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			defaultPort = p
		}
	}
	port := flag.Int("port", defaultPort, "Port to listen on")
	generateKey := flag.Bool("generate-key", false, "Generate a new private key and exit")
	flag.Parse()

	// Initialize logger
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	fmt.Printf("DEBUG: LOG_LEVEL from env: %s\n", os.Getenv("LOG_LEVEL"))

	// Set libp2p logging level based on environment variable
	libp2pLogLevel := os.Getenv("LIBP2P_LOGGING")
	fmt.Printf("DEBUG: LIBP2P_LOGGING from env: %s\n", libp2pLogLevel)
	if libp2pLogLevel != "" {
		switch libp2pLogLevel {
		case "debug":
			logging.SetAllLoggers(logging.LevelDebug)
		case "info":
			logging.SetAllLoggers(logging.LevelInfo)
		case "warn":
			logging.SetAllLoggers(logging.LevelWarn)
		case "error":
			logging.SetAllLoggers(logging.LevelError)
		case "fatal":
			logging.SetAllLoggers(logging.LevelFatal)
		default:
			log.Warnf("Unknown LIBP2P_LOGGING level: %s. Defaulting to info.", libp2pLogLevel)
			logging.SetAllLoggers(logging.LevelInfo)
		}
	} else {
		// Default libp2p logging to info if not specified
		logging.SetAllLoggers(logging.LevelInfo)
	}

	if *generateKey {
		// Generate a new Ed25519 private key
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			fmt.Printf("Error generating private key: %v\n", err)
			os.Exit(1)
		}

		// Get the Peer ID from the private key
		peerID, err := peer.IDFromPrivateKey(priv)
		if err != nil {
			fmt.Printf("Error getting Peer ID: %v\n", err)
			os.Exit(1)
		}

		// Encode the raw private key to hex for storage (64 bytes = 128 hex characters)
		rawPriv, err := priv.Raw()
		if err != nil {
			fmt.Printf("Error getting raw private key: %v\n", err)
			os.Exit(1)
		}
		privateKeyHex := hex.EncodeToString(rawPriv)

		// Construct a multiaddress (using a placeholder IP and default port 4001)
		multiAddrStr := fmt.Sprintf("/ip4/127.0.0.1/tcp/4001/p2p/%s", peerID.String())
		_, err = multiaddr.NewMultiaddr(multiAddrStr) // Validate multiaddress format
		if err != nil {
			fmt.Printf("Error creating multiaddress: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Generated Private Key (hex):", privateKeyHex)
		fmt.Println("Derived Peer ID:", peerID.String())
		fmt.Println("Expected Multiaddress (local placeholder):", multiAddrStr)
		fmt.Println("\nRemember to replace '127.0.0.1' with your bootstrap node's public IP address when configuring other nodes.")
		return
	}

	// Load config
	cfg := config.LoadConfig()

	// Create a context that is canceled on a graceful shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize and start the bootstrap node service
	node, err := service.NewBootstrapNode(ctx, *port, cfg)
	if err != nil {
		log.Fatalf("Failed to create bootstrap node: %v", err)
	}

	// Print the node's multiaddress for others to connect
	log.Infof("🚀 Bootstrap node started. ID: %s", node.Host.ID().String())
	log.Infof("🌍 Listening on addresses: %s", node.Host.Addrs())

	// Start periodic peer logging
	go func() {
		ticker := time.NewTicker(60 * time.Second) // Log every 10 seconds
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				peers := node.Host.Network().Peers()
				log.Infof("Connected peers: %d", len(peers))
				for _, p := range peers {
					log.Debugf("  - %s", p.String())
				}
			}
		}
	}()

	// Wait for a shutdown signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println()
	log.Info("Shutting down bootstrap node...")
	node.Host.Close()
}
