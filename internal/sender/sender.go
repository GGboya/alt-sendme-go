package sender

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"alt-sendme-go/internal/p2p"
	"alt-sendme-go/internal/ticket"
	"alt-sendme-go/internal/utils"

	"github.com/libp2p/go-libp2p/core/network"
)

// Sender 处理文件发送
type Sender struct {
	node     *p2p.Node
	filePath string
	fileHash string
	fileName string
	fileSize int64
	ctx      context.Context
	cancel   context.CancelFunc
}

// StartShare 启动文件分享
func StartShare(ctx context.Context, filePath string, port int, onProgress func(bytesTransferred, totalBytes int64, speed float64)) (*ticket.Ticket, error) {
	// 验证文件存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("path is a directory, only files are supported in v1")
	}

	// 计算文件哈希
	fileHash, err := utils.CalculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// 创建 P2P 节点
	node, err := p2p.NewNode(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create P2P node: %w", err)
	}

	// 获取节点地址
	peerAddr, err := node.GetPeerAddr()
	if err != nil {
		node.Close()
		return nil, fmt.Errorf("failed to get peer address: %w", err)
	}

	// 创建 sender
	senderCtx, cancel := context.WithCancel(ctx)
	sender := &Sender{
		node:     node,
		filePath: filePath,
		fileHash: fileHash,
		fileName: utils.GetFileName(filePath),
		fileSize: fileInfo.Size(),
		ctx:      senderCtx,
		cancel:   cancel,
	}

	// 设置流处理器
	node.SetStreamHandler(func(stream network.Stream) {
		defer stream.Close()
		sender.handleStream(stream, onProgress)
	})

	// 创建 ticket
	tkt := &ticket.Ticket{
		PeerAddr: peerAddr,
		FileHash: fileHash,
		FileName: sender.fileName,
		FileSize: sender.fileSize,
	}

	return tkt, nil
}

// handleStream 处理接收方的连接
func (s *Sender) handleStream(stream network.Stream, onProgress func(bytesTransferred, totalBytes int64, speed float64)) {
	// 创建文件元数据
	metadata := &p2p.FileMetadata{
		FileName: s.fileName,
		FileSize: s.fileSize,
		FileHash: s.fileHash,
	}

	// 发送元数据
	if err := p2p.SendMetadata(stream, metadata); err != nil {
		fmt.Printf("Error sending metadata: %v\n", err)
		return
	}

	// 打开文件
	file, err := os.Open(s.filePath)
	if err != nil {
		p2p.SendError(stream, fmt.Errorf("failed to open file: %w", err))
		return
	}
	defer file.Close()

	// 分块发送文件
	buffer := make([]byte, p2p.ChunkSize)
	startTime := time.Now()
	var bytesSent int64

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// 读取文件块
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			p2p.SendError(stream, fmt.Errorf("failed to read file: %w", err))
			return
		}

		// 发送文件块
		if err := p2p.SendChunk(stream, buffer[:n]); err != nil {
			fmt.Printf("Error sending chunk: %v\n", err)
			return
		}

		bytesSent += int64(n)

		// 调用进度回调
		if onProgress != nil {
			elapsed := time.Since(startTime)
			speed := utils.CalculateSpeed(bytesSent, elapsed)
			onProgress(bytesSent, s.fileSize, speed)
		}
	}

	// 发送完成消息
	if err := p2p.SendDone(stream); err != nil {
		fmt.Printf("Error sending done message: %v\n", err)
		return
	}

	fmt.Printf("File transfer completed: %d bytes sent\n", bytesSent)
}

// Stop 停止分享
func (s *Sender) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.node != nil {
		s.node.Close()
	}
}


