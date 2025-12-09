package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/multiformats/go-multiaddr"
)

const (
	// ProtocolID 是文件传输协议标识符
	ProtocolID = protocol.ID("/sendme/file-transfer/1.0.0")
	// ServiceTag 用于 mDNS 服务发现
	ServiceTag = "sendme"
)

// Node 封装了 libp2p Host 和相关功能
type Node struct {
	host.Host
	ctx    context.Context
	cancel context.CancelFunc
}

// NewNode 创建一个新的 P2P 节点
func NewNode(ctx context.Context, port int) (*Node, error) {
	// 生成随机密钥对
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	nodeCtx, cancel := context.WithCancel(ctx)

	// 创建一个变量来存储 host，以便在 Peer Source 中使用
	var h host.Host

	// 创建 Peer Source 函数，从已连接的 peers 中发现中继服务器
	// 使用闭包捕获 host 引用
	// 注意：这个函数在节点刚创建时可能返回空结果，因为还没有连接的 peers
	// AutoRelay 会定期调用这个函数来发现中继服务器
	peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		r := make(chan peer.AddrInfo, numPeers)
		go func() {
			defer close(r)
			if h == nil {
				return
			}
			// 从已连接的 peers 中查找支持中继的节点
			peers := h.Network().Peers()
			if len(peers) == 0 {
				// 如果没有已连接的 peers，返回空结果
				// AutoRelay 会通过其他机制（如 DHT）发现中继
				return
			}

			for _, peerID := range peers {
				select {
				case <-ctx.Done():
					return
				default:
					// 检查这个 peer 是否支持中继协议
					protos, err := h.Peerstore().GetProtocols(peerID)
					if err == nil {
						for _, proto := range protos {
							// 检查是否支持 circuit relay
							if proto == "/libp2p/circuit/relay/0.2.0" || proto == "/p2p-circuit" {
								addrs := h.Peerstore().Addrs(peerID)
								if len(addrs) > 0 {
									select {
									case r <- peer.AddrInfo{ID: peerID, Addrs: addrs}:
									case <-ctx.Done():
										return
									}
									break
								}
							}
						}
					}
				}
			}
		}()
		return r
	}

	// 创建 libp2p Host，启用 AutoRelay
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
		// 启用 Circuit Relay v2 客户端支持
		libp2p.EnableRelay(),
		// 启用 AutoRelay，使用我们创建的 Peer Source
		libp2p.EnableAutoRelay(
			autorelay.WithPeerSource(peerSource),
		),
	}

	h, err = libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	node := &Node{
		Host:   h,
		ctx:    nodeCtx,
		cancel: cancel,
	}

	// 启动 mDNS 服务发现
	if err := node.startMDNS(); err != nil {
		cancel()
		h.Close()
		return nil, fmt.Errorf("failed to start mDNS: %w", err)
	}

	// 等待一段时间让节点发现中继服务器
	// 这有助于确保在获取地址时已经连接到中继
	go func() {
		time.Sleep(2 * time.Second)
		// 触发 AutoRelay 发现中继服务器
		_ = h.Network().Peers()
	}()

	return node, nil
}

// startMDNS 启动 mDNS 服务发现
func (n *Node) startMDNS() error {
	service := mdns.NewMdnsService(n.Host, ServiceTag, &mdnsNotifee{h: n.Host})
	return service.Start()
}

// mdnsNotifee 实现 mdns.Notifee 接口
type mdnsNotifee struct {
	h host.Host
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if err := n.h.Connect(context.Background(), pi); err != nil {
		// 静默处理连接错误，可能是正常的
	}
}

// SetStreamHandler 设置流处理器
func (n *Node) SetStreamHandler(handler func(network.Stream)) {
	n.Host.SetStreamHandler(ProtocolID, handler)
}

// ConnectToPeer 连接到指定的 peer
// libp2p 的 AutoRelay 会自动处理中继连接：
// 1. 首先尝试直连
// 2. 如果直连失败（例如在 NAT 后面），自动使用中继服务器
func (n *Node) ConnectToPeer(ctx context.Context, peerAddr string) error {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	// 等待一段时间，让节点有机会发现和连接到中继服务器
	// 这对于跨网络连接很重要
	// 同时检查是否有可用的中继连接
	fmt.Printf("Connecting to peer... (this may take a few seconds for relay discovery)\n")
	time.Sleep(5 * time.Second)

	// 设置连接超时，给足够的时间让中继连接建立
	connectCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// libp2p 的 Connect 会自动处理：
	// - 直连尝试
	// - 如果失败，自动通过 AutoRelay 使用中继服务器
	// 但我们需要确保接收端也有中继连接可用

	// 检查是否有可用的中继连接
	connectedPeers := n.Host.Network().Peers()
	fmt.Printf("Current connected peers: %d\n", len(connectedPeers))

	if err := n.Host.Connect(connectCtx, *info); err != nil {
		fmt.Printf("Direct connection failed: %v\n", err)
		fmt.Printf("Attempting relay connection...\n")

		// 如果连接失败，尝试通过中继连接
		// 检查是否有可用的中继连接
		relayAddrs := n.findRelayAddresses(info.ID)
		if len(relayAddrs) > 0 {
			fmt.Printf("Found %d relay addresses, attempting connection...\n", len(relayAddrs))
			// 尝试通过中继连接
			relayInfo := peer.AddrInfo{
				ID:    info.ID,
				Addrs: relayAddrs,
			}
			if relayErr := n.Host.Connect(connectCtx, relayInfo); relayErr != nil {
				return fmt.Errorf("failed to connect to peer (tried direct and relay): direct=%v, relay=%v. Make sure the sender is still running.", err, relayErr)
			}
			fmt.Printf("Connected via relay!\n")
			return nil
		}

		// 如果没有找到中继，提供更详细的错误信息
		return fmt.Errorf("failed to connect to peer: %w. Possible reasons: 1) Sender is not running, 2) Network connectivity issues, 3) No relay servers available. Make sure the sender is still running and try again.", err)
	}

	fmt.Printf("Connected successfully!\n")

	return nil
}

// findRelayAddresses 查找可用的中继地址
func (n *Node) findRelayAddresses(targetPeer peer.ID) []multiaddr.Multiaddr {
	var relayAddrs []multiaddr.Multiaddr

	// 从已连接的 peers 中查找中继服务器
	for _, peerID := range n.Host.Network().Peers() {
		// 检查这个 peer 是否支持中继协议
		protos, err := n.Host.Peerstore().GetProtocols(peerID)
		if err == nil {
			for _, proto := range protos {
				// 检查是否支持 circuit relay
				if proto == "/libp2p/circuit/relay/0.2.0" {
					// 构建通过中继的地址
					addrs := n.Host.Peerstore().Addrs(peerID)
					for _, addr := range addrs {
						// 构建 /p2p-circuit 地址
						relayAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("%s/p2p/%s/p2p-circuit/p2p/%s",
							addr.String(), peerID.String(), targetPeer.String()))
						if err == nil {
							relayAddrs = append(relayAddrs, relayAddr)
						}
					}
				}
			}
		}
	}

	return relayAddrs
}

// GetPeerAddr 获取节点的完整地址（用于生成 ticket）
// libp2p 的 AutoRelay 会自动处理中继连接，所以这里返回一个可用的地址即可
// 连接时会自动尝试直连，失败时自动使用中继
func (n *Node) GetPeerAddr() (string, error) {
	// 等待一小段时间，让 AutoRelay 有机会发现中继服务器
	// 这样地址列表可能包含更多信息
	time.Sleep(2 * time.Second)

	addrs := n.Host.Addrs()
	if len(addrs) == 0 {
		return "", fmt.Errorf("no addresses available")
	}

	// 使用第一个可用地址
	// libp2p 的 Connect() 方法会自动处理中继连接
	addr := addrs[0]
	peerAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", addr, n.Host.ID()))
	if err != nil {
		return "", fmt.Errorf("failed to create peer address: %w", err)
	}

	return peerAddr.String(), nil
}

// ParsePeerIDFromAddr 从 multiaddr 字符串中解析 peer ID
func ParsePeerIDFromAddr(peerAddr string) (peer.ID, error) {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return "", fmt.Errorf("invalid multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return "", fmt.Errorf("failed to parse peer info: %w", err)
	}

	return info.ID, nil
}

// Close 关闭节点
func (n *Node) Close() error {
	n.cancel()
	return n.Host.Close()
}
