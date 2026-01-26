package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"submissions-bootstrap-node/pkg/config"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/powerloom/snapshot-sequencer-validator/pkgs/gossipconfig"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

// BootstrapNode struct holds the libp2p host and the DHT
type BootstrapNode struct {
	Host   host.Host
	Pubsub *pubsub.PubSub
	DHT    *dht.IpfsDHT
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBootstrapNode creates and initializes a new libp2p host configured as a bootstrap node
func NewBootstrapNode(ctx context.Context, port int, cfg config.Config) (*BootstrapNode, error) {
	// Create cancelable context for this bootstrap node
	hostCtx, cancel := context.WithCancel(ctx)
	var priv crypto.PrivKey
	var err error

	if cfg.PrivateKey != "" {
		privBytes, err := hex.DecodeString(cfg.PrivateKey)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to decode private key: %w", err)
		}
		priv, err = crypto.UnmarshalEd25519PrivateKey(privBytes)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
		}
	} else {
		priv, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
	}

	// 1. Create a new resource manager with custom limits.
	scalingLimits := rcmgr.DefaultLimits
	limitsCfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			StreamsOutbound: rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			Streams:         rcmgr.Unlimited,
			Conns:           rcmgr.Unlimited,
			ConnsOutbound:   rcmgr.Unlimited,
			ConnsInbound:    rcmgr.Unlimited,
			FD:              rcmgr.Unlimited,
			Memory:          rcmgr.LimitVal64(rcmgr.Unlimited),
		},
		Transient: rcmgr.ResourceLimits{
			StreamsOutbound: rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			Streams:         rcmgr.Unlimited,
			Conns:           rcmgr.Unlimited,
			ConnsOutbound:   rcmgr.Unlimited,
			ConnsInbound:    rcmgr.Unlimited,
			FD:              rcmgr.Unlimited,
			Memory:          rcmgr.LimitVal64(rcmgr.Unlimited),
		},
	}
	limiter := rcmgr.NewFixedLimiter(limitsCfg.Build(scalingLimits.AutoScale()))
	rscMgr, err := rcmgr.NewResourceManager(limiter, rcmgr.WithMetricsDisabled())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource manager: %w", err)
	}

	// Create a connection manager.
	connMgr, err := connmgr.NewConnManager(
		cfg.ConnManagerLowWater,  // Lowwater
		cfg.ConnManagerHighWater, // Highwater
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	var kadDHT *dht.IpfsDHT
	var notificationBundle *network.NotifyBundle
	// Create the libp2p host options
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.ResourceManager(rscMgr),
		libp2p.ConnectionManager(connMgr),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			kadDHT, err = dht.New(hostCtx, h, dht.Mode(dht.ModeServer))
			return kadDHT, err
		}),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Transport(tcp.NewTCPTransport),
	}

	// Add public IP address if configured
	if cfg.PublicIP != "" {
		publicAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.PublicIP, port))
		if err != nil {
			log.Errorf("Failed to create public multiaddr: %v", err)
		} else {
			opts = append(opts, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
				return append(addrs, publicAddr)
			}))
		}
	}

	// Create the libp2p host with the DHT in server mode.
	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Get standardized gossipsub parameters for consistency across network
	gossipParams, peerScoreParams, peerScoreThresholds, paramHash := gossipconfig.ConfigureSnapshotSubmissionsMesh(h.ID())

	// Create a new GossipSub instance with standardized parameters
	gs, err := pubsub.NewGossipSub(hostCtx, h,
		pubsub.WithGossipSubParams(*gossipParams),
		pubsub.WithPeerScore(peerScoreParams, peerScoreThresholds),
		pubsub.WithFloodPublish(true),
		pubsub.WithMessageSignaturePolicy(pubsub.StrictSign),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}
	
	log.Infof("🔑 Gossipsub parameter hash: %s (bootstrap node)", paramHash)

	log.Infof("Libp2p host created with ID: %s, listening on: %v", h.ID(), h.Addrs())
	log.Infof("Bootstrap node DHT routing table size: %d", kadDHT.RoutingTable().Size())
	log.Infof("Bootstrap node created with ID: %s, listening on: %v", h.ID(), h.Addrs())

	// Store notification bundle for cleanup
	notificationBundle = &network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			log.Infof("Bootstrap Peer connected: %s, Addr: %s", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
		DisconnectedF: func(_ network.Network, conn network.Conn) {
			log.Infof("Bootstrap Peer disconnected: %s, Addr: %s", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
	}
	h.Network().Notify(notificationBundle)

	return &BootstrapNode{
		Host:   h,
		Pubsub: gs,
		DHT:    kadDHT,
		ctx:    hostCtx,
		cancel: cancel,
	}, nil
}

// Close closes the bootstrap node and releases all resources
func (n *BootstrapNode) Close() error {
	if n.cancel != nil {
		n.cancel()
	}
	return n.Host.Close()
}
