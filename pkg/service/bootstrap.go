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
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

// BootstrapNode is a discovery-only libp2p entry point: stable peer ID, DHT routing,
// and optional circuit relay. It does not run gossipsub — mesh traffic stays on
// snapshotters and validators.
type BootstrapNode struct {
	Host               host.Host
	DHT                *dht.IpfsDHT
	notificationBundle *network.NotifyBundle
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewBootstrapNode creates and initializes a new libp2p host configured as a bootstrap node.
func NewBootstrapNode(ctx context.Context, port int, cfg config.Config) (*BootstrapNode, error) {
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

	if cfg.ConnManagerHighWater < cfg.ConnManagerLowWater {
		return nil, fmt.Errorf("CONN_MANAGER_HIGH_WATER (%d) must be >= CONN_MANAGER_LOW_WATER (%d)",
			cfg.ConnManagerHighWater, cfg.ConnManagerLowWater)
	}

	connLimit := cfg.ConnManagerHighWater + 50
	memoryLimit := int64(cfg.RcmgrMemoryLimitMB) << 20

	scalingLimits := rcmgr.DefaultLimits
	libp2p.SetDefaultServiceLimits(&scalingLimits)
	limitsCfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			Conns:           rcmgr.LimitVal(connLimit),
			ConnsInbound:    rcmgr.LimitVal(connLimit),
			ConnsOutbound:   rcmgr.LimitVal(connLimit),
			Streams:         rcmgr.LimitVal(4096),
			StreamsInbound:  rcmgr.LimitVal(2048),
			StreamsOutbound: rcmgr.LimitVal(2048),
			Memory:          rcmgr.LimitVal64(memoryLimit),
			FD:              rcmgr.LimitVal(connLimit * 4),
		},
		Transient: rcmgr.ResourceLimits{
			Conns:           rcmgr.LimitVal(connLimit),
			ConnsInbound:    rcmgr.LimitVal(connLimit),
			ConnsOutbound:   rcmgr.LimitVal(connLimit),
			Streams:         rcmgr.LimitVal(1024),
			StreamsInbound:  rcmgr.LimitVal(512),
			StreamsOutbound: rcmgr.LimitVal(512),
			Memory:          rcmgr.LimitVal64(memoryLimit / 4),
			FD:              rcmgr.LimitVal(connLimit * 2),
		},
	}
	limiter := rcmgr.NewFixedLimiter(limitsCfg.Build(scalingLimits.AutoScale()))
	rscMgr, err := rcmgr.NewResourceManager(limiter, rcmgr.WithMetricsDisabled())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource manager: %w", err)
	}

	connMgr, err := connmgr.NewConnManager(
		cfg.ConnManagerLowWater,
		cfg.ConnManagerHighWater,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	var kadDHT *dht.IpfsDHT
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.ResourceManager(rscMgr),
		libp2p.ConnectionManager(connMgr),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			kadDHT, err = dht.New(hostCtx, h, dht.Mode(dht.ModeServer))
			return kadDHT, err
		}),
		libp2p.ForceReachabilityPublic(),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Transport(tcp.NewTCPTransport),
	}
	if cfg.EnableRelayService {
		relayResources := relay.DefaultResources()
		relayResources.MaxReservations = cfg.RelayMaxReservations
		relayResources.MaxReservationsPerIP = cfg.RelayMaxReservationsPerIP
		relayResources.MaxCircuits = cfg.RelayMaxCircuits
		opts = append(opts, libp2p.EnableRelayService(relay.WithResources(relayResources)))
		log.Infof("Circuit relay enabled (max_reservations=%d per_ip=%d max_circuits=%d)",
			cfg.RelayMaxReservations, cfg.RelayMaxReservationsPerIP, cfg.RelayMaxCircuits)
	}

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

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	log.Infof("Libp2p bootstrap host ID: %s, listening on: %v", h.ID(), h.Addrs())
	log.Infof("DHT routing table size: %d (relay=%t conn_high=%d memory_limit_mb=%d)",
		kadDHT.RoutingTable().Size(), cfg.EnableRelayService, cfg.ConnManagerHighWater, cfg.RcmgrMemoryLimitMB)

	var notificationBundle *network.NotifyBundle
	if cfg.LogPeerConnections {
		notificationBundle = &network.NotifyBundle{
			ConnectedF: func(_ network.Network, conn network.Conn) {
				log.Infof("Peer connected: %s, addr: %s", conn.RemotePeer(), conn.RemoteMultiaddr())
			},
			DisconnectedF: func(_ network.Network, conn network.Conn) {
				log.Infof("Peer disconnected: %s, addr: %s", conn.RemotePeer(), conn.RemoteMultiaddr())
			},
		}
		h.Network().Notify(notificationBundle)
	}

	node := &BootstrapNode{
		Host:               h,
		DHT:                kadDHT,
		notificationBundle: notificationBundle,
		ctx:                hostCtx,
		cancel:             cancel,
	}

	go node.startPeerstoreGC()

	return node, nil
}

// Close closes the bootstrap node and releases all resources.
func (n *BootstrapNode) Close() error {
	var errs []error

	if n.cancel != nil {
		n.cancel()
	}

	if n.notificationBundle != nil && n.Host != nil {
		n.Host.Network().StopNotify(n.notificationBundle)
	}

	if n.DHT != nil {
		if err := n.DHT.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close DHT: %w", err))
		}
	}

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
			}
		}
	}
}
