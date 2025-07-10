package service

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	log "github.com/sirupsen/logrus"
)

// BootstrapNode struct holds the libp2p host
type BootstrapNode struct {
	Host host.Host
}

// NewBootstrapNode creates and initializes a new libp2p host configured as a bootstrap node
func NewBootstrapNode(ctx context.Context, port int) (*BootstrapNode, error) {
	// Create a connection manager
	connMgr, err := connmgr.NewConnManager(
		100, // Lowwater
		400, // Highwater
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Create the libp2p host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.ConnectionManager(connMgr),
		libp2p.ForceReachabilityPublic(), // Announce ourselves as publicly reachable
		libp2p.NATPortMap(),              // Attempt to open ports via NAT
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	log.Infof("Libp2p host created with ID: %s", h.ID().String())

	return &BootstrapNode{
		Host: h,
	}, nil
}
