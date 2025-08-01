package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	log "github.com/sirupsen/logrus"
)

// BootstrapNode struct holds the libp2p host and the DHT
type BootstrapNode struct {
	Host host.Host
	DHT  *dht.IpfsDHT
}

// NewBootstrapNode creates and initializes a new libp2p host configured as a bootstrap node
func NewBootstrapNode(ctx context.Context, port int, privateKeyHex string) (*BootstrapNode, error) {
	var priv crypto.PrivKey
	var err error

	if privateKeyHex != "" {
		privBytes, err := hex.DecodeString(privateKeyHex)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key: %w", err)
		}
		priv, err = crypto.UnmarshalEd25519PrivateKey(privBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
		}
	} else {
		priv, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
	}

	// 1. Create a new resource manager with custom limits.
	scalingLimits := rcmgr.DefaultLimits
	cfg := rcmgr.PartialLimitConfig{
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
	limiter := rcmgr.NewFixedLimiter(cfg.Build(scalingLimits.AutoScale()))
	rscMgr, err := rcmgr.NewResourceManager(limiter, rcmgr.WithMetricsDisabled())
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager: %w", err)
	}

	// Create a connection manager.
	connMgr, err := connmgr.NewConnManager(
		100, // Lowwater
		400, // Highwater
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	var kadDHT *dht.IpfsDHT
	// Create the libp2p host with the DHT in server mode.
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.ResourceManager(rscMgr),
		libp2p.ConnectionManager(connMgr),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			kadDHT, err = dht.New(ctx, h, dht.Mode(dht.ModeServer))
			return kadDHT, err
		}),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Transport(tcp.NewTCPTransport),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Create a new GossipSub instance
	_, err = pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	log.Infof("Libp2p host created with ID: %s", h.ID().String())

	return &BootstrapNode{
		Host: h,
		DHT:  kadDHT,
	}, nil
}
