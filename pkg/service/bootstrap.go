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
	"github.com/libp2p/go-libp2p/core/peer"
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
	Host              host.Host
	Pubsub            *pubsub.PubSub
	DHT               *dht.IpfsDHT
	notificationBundle *network.NotifyBundle
	ctx               context.Context
	cancel            context.CancelFunc
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
	// BOUND connections to match connection manager limits to prevent unbounded peer scoring
	// Peer scoring tracks state per connection, so bounding connections bounds peer score memory
	scalingLimits := rcmgr.DefaultLimits
	limitsCfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			StreamsOutbound: rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			Streams:         rcmgr.Unlimited,
			// Bound connections to connection manager high water mark + buffer
			// This ensures peer scoring state is bounded
			Conns:         rcmgr.LimitVal(cfg.ConnManagerHighWater + 100), // Small buffer for transient connections
			ConnsOutbound: rcmgr.LimitVal(cfg.ConnManagerHighWater + 100),
			ConnsInbound:  rcmgr.LimitVal(cfg.ConnManagerHighWater + 100),
			FD:            rcmgr.Unlimited,
			Memory:       rcmgr.LimitVal64(rcmgr.Unlimited),
		},
		Transient: rcmgr.ResourceLimits{
			StreamsOutbound: rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			Streams:         rcmgr.Unlimited,
			Conns:           rcmgr.LimitVal(cfg.ConnManagerHighWater + 100),
			ConnsOutbound:   rcmgr.LimitVal(cfg.ConnManagerHighWater + 100),
			ConnsInbound:    rcmgr.LimitVal(cfg.ConnManagerHighWater + 100),
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

	// Create GossipSub with peer scoring for DDoS protection
	// Peer scoring is bounded by resource manager connection limits (matching connection manager)
	// Connection limits ensure peer score state cannot grow unbounded
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

	node := &BootstrapNode{
		Host:              h,
		Pubsub:            gs,
		DHT:               kadDHT,
		notificationBundle: notificationBundle,
		ctx:               hostCtx,
		cancel:            cancel,
	}

	go node.startPeerstoreGC()

	return node, nil
}

// Close closes the bootstrap node and releases all resources
func (n *BootstrapNode) Close() error {
	var errs []error
	
	// Cancel context first to stop all background operations
	if n.cancel != nil {
		n.cancel()
	}
	
	// Unregister notification bundle to prevent memory leaks
	if n.notificationBundle != nil && n.Host != nil {
		n.Host.Network().StopNotify(n.notificationBundle)
	}
	
	// Close DHT to release routing table and provider storage
	if n.DHT != nil {
		if err := n.DHT.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close DHT: %w", err))
		}
	}
	
	// Close the host (this will close all connections and clean up resources)
	if n.Host != nil {
		if err := n.Host.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close host: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	return nil
}

// startPeerstoreGC periodically removes stale peers from the peerstore.
// Disconnected peers are cleaned immediately: ClearAddrs is called first
// (RemovePeer does not clear addresses per the libp2p interface contract),
// then RemovePeer removes keybook/protobook/metadata entries.
func (n *BootstrapNode) startPeerstoreGC() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			peers := n.Host.Peerstore().Peers()
			connectedPeers := n.Host.Network().Peers()
			connectedSet := make(map[peer.ID]struct{}, len(connectedPeers))
			for _, p := range connectedPeers {
				connectedSet[p] = struct{}{}
			}
			removed := 0
			for _, p := range peers {
				if p == n.Host.ID() {
					continue
				}
				if _, connected := connectedSet[p]; connected {
					continue
				}
				n.Host.Peerstore().ClearAddrs(p)
				n.Host.Peerstore().RemovePeer(p)
				removed++
			}
			remaining := len(n.Host.Peerstore().Peers())
			if removed > 0 {
				log.Infof("Peerstore GC: removed %d stale peers, %d remaining (connected: %d)", removed, remaining, len(connectedPeers))
			} else {
				log.Debugf("Peerstore GC: no stale peers removed, %d in store (connected: %d)", remaining, len(connectedPeers))
			}
		}
	}
}
