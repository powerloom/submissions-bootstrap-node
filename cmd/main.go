package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"submissions-bootstrap-node/pkg/service"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	// Command-line flags
	port := flag.Int("port", 4001, "Port to listen on")
	flag.Parse()

	// Initialize logger
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.InfoLevel)

	// Create a context that is canceled on a graceful shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize and start the bootstrap node service
	node, err := service.NewBootstrapNode(ctx, *port)
	if err != nil {
		log.Fatalf("Failed to create bootstrap node: %v", err)
	}

	// Print the node's multiaddress for others to connect
	log.Infof("🚀 Bootstrap node started. ID: %s", node.Host.ID().String())
	log.Infof("🌍 Listening on addresses: %s", node.Host.Addrs())

	// Wait for a shutdown signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println()
	log.Info("Shutting down bootstrap node...")
	node.Host.Close()
}
