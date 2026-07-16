package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

var (
	GlobalHost host.Host
	GlobalDHT  *dht.IpfsDHT
)

// InitHost starts the libp2p node and connects to the global mesh.
func InitHost(ctx context.Context) error {
	if GlobalIdentity == nil {
		return fmt.Errorf("p2p identity not initialized")
	}

	cfg := config.Cfg.P2P
	if !cfg.Enabled {
		return nil
	}

	port := cfg.ListenPort
	if port == 0 {
		port = 7479
	}

	// 1. Connection Manager (prevent resource exhaustion)
	maxPeers := cfg.MaxPeers
	if maxPeers == 0 {
		maxPeers = 100
	}
	cm, err := connmgr.NewConnManager(
		maxPeers/2, // Low watermark
		maxPeers,   // High watermark
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to create conn manager: %w", err)
	}

	// 2. Setup options
	opts := []libp2p.Option{
		libp2p.Identity(GlobalIdentity.PrivKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port),
		),
		libp2p.ConnectionManager(cm),
		libp2p.DefaultTransports,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
		libp2p.NATPortMap(), // Attempt to open ports via UPnP
	}

	if cfg.EnableRelay {
		opts = append(opts, libp2p.EnableRelay(), libp2p.EnableAutoRelayWithStaticRelays(nil)) // Enable relay client
	}

	// 3. Create the Host
	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %w", err)
	}
	GlobalHost = h
	logging.Logger.Info("P2P Host started", "peer_id", h.ID(), "addrs", h.Addrs())

	// 4. Initialize DHT
	kademliaDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeAuto))
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}
	GlobalDHT = kademliaDHT

	// Bootstrap the DHT
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	var wg sync.WaitGroup
	bootstrapPeers := cfg.BootstrapPeers
	if len(bootstrapPeers) == 0 {
		// Default hardcoded IPFS bootstrap nodes for now, 
		// ideally we will use our own tkngate official nodes later.
		bootstrapPeers = []string{
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJVtxdv6PtBa",
		}
	}

	connectedCount := 0
	for _, addr := range bootstrapPeers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			logging.Logger.Warn("Invalid bootstrap address", "addr", addr, "error", err)
			continue
		}
		peerinfo, _ := peer.AddrInfoFromP2pAddr(ma)
		if peerinfo == nil {
			continue
		}
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			if err := h.Connect(ctx, pi); err != nil {
				logging.Logger.Debug("Failed to connect to bootstrap peer", "peer", pi.ID, "error", err)
			} else {
				logging.Logger.Info("Connected to bootstrap peer", "peer", pi.ID)
				connectedCount++
			}
		}(*peerinfo)
	}
	wg.Wait()

	// 5. Setup Discovery
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, "tkngate-mesh-v1")

	// Local mDNS discovery
	if cfg.EnableMDNS {
		mdnsService := mdns.NewMdnsService(h, "tkngate-mesh-local", &mdnsNotifee{h: h, ctx: ctx})
		if err := mdnsService.Start(); err != nil {
			logging.Logger.Warn("Failed to start mDNS", "error", err)
		} else {
			logging.Logger.Info("mDNS Discovery started")
		}
	}

	// Find peers asynchronously
	go func() {
		for {
			peerChan, err := routingDiscovery.FindPeers(ctx, "tkngate-mesh-v1")
			if err != nil {
				time.Sleep(30 * time.Second)
				continue
			}

			for p := range peerChan {
				if p.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, _ = h.Network().DialPeer(ctx, p.ID)
				}
			}
			time.Sleep(1 * time.Minute)
		}
	}()

	return nil
}

type mdnsNotifee struct {
	h   host.Host
	ctx context.Context
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if n.h.Network().Connectedness(pi.ID) != network.Connected {
		logging.Logger.Debug("mDNS found peer, connecting...", "peer", pi.ID)
		_ = n.h.Connect(n.ctx, pi)
	}
}
