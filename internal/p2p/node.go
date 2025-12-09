package p2p

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
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

	// 创建 libp2p Host
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	nodeCtx, cancel := context.WithCancel(ctx)

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
func (n *Node) ConnectToPeer(ctx context.Context, peerAddr string) error {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	if err := n.Host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	return nil
}

// GetPeerAddr 获取节点的完整地址（用于生成 ticket）
func (n *Node) GetPeerAddr() (string, error) {
	addrs := n.Host.Addrs()
	if len(addrs) == 0 {
		return "", fmt.Errorf("no addresses available")
	}

	// 使用第一个地址
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

