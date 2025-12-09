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
	peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		r := make(chan peer.AddrInfo)
		go func() {
			defer close(r)
			if h == nil {
				return
			}
			// 从已连接的 peers 中查找支持中继的节点
			for _, peerID := range h.Network().Peers() {
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

	// 设置连接超时
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// libp2p 的 Connect 会自动处理：
	// - 直连尝试
	// - 如果失败，自动通过 AutoRelay 使用中继服务器
	if err := n.Host.Connect(connectCtx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer (tried direct and relay): %w", err)
	}

	return nil
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
