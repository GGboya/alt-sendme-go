package receiver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"alt-sendme-go/internal/p2p"
	"alt-sendme-go/internal/ticket"
	"alt-sendme-go/internal/utils"

	"github.com/libp2p/go-libp2p/core/host"
)

// Receiver 处理文件接收
type Receiver struct {
	node       host.Host
	outputPath string
	ctx        context.Context
}

// Receive 接收文件
func Receive(ctx context.Context, tkt *ticket.Ticket, outputPath string, port int, onProgress func(bytesTransferred, totalBytes int64, speed float64)) error {
	// 创建 P2P 节点
	node, err := p2p.NewNode(ctx, port)
	if err != nil {
		return fmt.Errorf("failed to create P2P node: %w", err)
	}
	defer node.Close()

	// 连接到发送方
	if err := node.ConnectToPeer(ctx, tkt.PeerAddr); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	// 从 ticket 中解析 peer ID
	peerID, err := p2p.ParsePeerIDFromAddr(tkt.PeerAddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer ID: %w", err)
	}

	// 打开流
	stream, err := node.NewStream(ctx, peerID, p2p.ProtocolID)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// 接收元数据
	metadata, err := p2p.ReceiveMetadata(stream)
	if err != nil {
		return fmt.Errorf("failed to receive metadata: %w", err)
	}

	// 验证文件哈希是否匹配
	if metadata.FileHash != tkt.FileHash {
		return fmt.Errorf("file hash mismatch: expected %s, got %s", tkt.FileHash, metadata.FileHash)
	}

	// 确定输出文件路径
	var outputFilePath string
	if outputPath != "" {
		// 如果指定了目录，在目录中创建文件
		if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
			outputFilePath = filepath.Join(outputPath, metadata.FileName)
		} else {
			// 如果是指定文件路径，直接使用
			outputFilePath = outputPath
		}
	} else {
		// 使用当前目录
		outputFilePath = metadata.FileName
	}

	// 检查文件是否已存在
	if _, err := os.Stat(outputFilePath); err == nil {
		return fmt.Errorf("file already exists: %s", outputFilePath)
	}

	// 确保输出目录存在
	if err := utils.EnsureDir(filepath.Dir(outputFilePath)); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 创建输出文件
	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// 创建哈希计算器用于验证
	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)

	// 接收文件数据
	startTime := time.Now()
	var bytesReceived int64

	for {
		chunk, err := p2p.ReceiveChunk(stream)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to receive chunk: %w", err)
		}

		// 写入文件
		if _, err := multiWriter.Write(chunk); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		bytesReceived += int64(len(chunk))

		// 调用进度回调
		if onProgress != nil {
			elapsed := time.Since(startTime)
			speed := utils.CalculateSpeed(bytesReceived, elapsed)
			onProgress(bytesReceived, metadata.FileSize, speed)
		}
	}

	// 验证文件完整性
	calculatedHash := fmt.Sprintf("%x", hash.Sum(nil))
	if calculatedHash != metadata.FileHash {
		os.Remove(outputFilePath)
		return fmt.Errorf("file integrity check failed: expected %s, got %s", metadata.FileHash, calculatedHash)
	}

	fmt.Printf("File received successfully: %s (%d bytes)\n", outputFilePath, bytesReceived)
	return nil
}

