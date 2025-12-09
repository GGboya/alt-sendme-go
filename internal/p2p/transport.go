package p2p

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

const (
	// ChunkSize 是文件传输的块大小（1MB）
	ChunkSize = 1024 * 1024
	// ReadTimeout 是读取超时时间
	ReadTimeout = 30 * time.Second
	// WriteTimeout 是写入超时时间
	WriteTimeout = 30 * time.Second
)

// FileMetadata 包含文件的元数据信息
type FileMetadata struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	FileHash string `json:"file_hash"`
}

// MessageType 定义消息类型
type MessageType uint8

const (
	MessageTypeMetadata MessageType = iota + 1
	MessageTypeChunk
	MessageTypeError
	MessageTypeDone
)

// Message 是传输消息的结构
type Message struct {
	Type     MessageType `json:"type"`
	Metadata *FileMetadata
	Data     []byte `json:"data,omitempty"`
	Error    string `json:"error,omitempty"`
}

// SendMetadata 发送文件元数据
func SendMetadata(stream network.Stream, metadata *FileMetadata) error {
	msg := Message{
		Type:     MessageTypeMetadata,
		Metadata: metadata,
	}

	return sendMessage(stream, &msg)
}

// ReceiveMetadata 接收文件元数据
func ReceiveMetadata(stream network.Stream) (*FileMetadata, error) {
	msg, err := receiveMessage(stream)
	if err != nil {
		return nil, err
	}

	if msg.Type != MessageTypeMetadata {
		return nil, fmt.Errorf("expected metadata message, got %d", msg.Type)
	}

	return msg.Metadata, nil
}

// SendChunk 发送文件块
func SendChunk(stream network.Stream, data []byte) error {
	msg := Message{
		Type: MessageTypeChunk,
		Data: data,
	}
	return sendMessage(stream, &msg)
}

// ReceiveChunk 接收文件块
func ReceiveChunk(stream network.Stream) ([]byte, error) {
	msg, err := receiveMessage(stream)
	if err != nil {
		return nil, err
	}

	if msg.Type == MessageTypeError {
		return nil, fmt.Errorf("remote error: %s", msg.Error)
	}

	if msg.Type == MessageTypeDone {
		return nil, io.EOF
	}

	if msg.Type != MessageTypeChunk {
		return nil, fmt.Errorf("expected chunk message, got %d", msg.Type)
	}

	return msg.Data, nil
}

// SendDone 发送完成消息
func SendDone(stream network.Stream) error {
	msg := Message{
		Type: MessageTypeDone,
	}
	return sendMessage(stream, &msg)
}

// SendError 发送错误消息
func SendError(stream network.Stream, err error) error {
	msg := Message{
		Type:  MessageTypeError,
		Error: err.Error(),
	}
	return sendMessage(stream, &msg)
}

// sendMessage 发送消息到流
func sendMessage(stream network.Stream, msg *Message) error {
	stream.SetWriteDeadline(time.Now().Add(WriteTimeout))
	defer stream.SetWriteDeadline(time.Time{})

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 先发送消息长度（4字节）
	length := uint32(len(data))
	if err := binary.Write(stream, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// 发送消息内容
	if _, err := stream.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// receiveMessage 从流接收消息
func receiveMessage(stream network.Stream) (*Message, error) {
	stream.SetReadDeadline(time.Now().Add(ReadTimeout))
	defer stream.SetReadDeadline(time.Time{})

	// 读取消息长度
	var length uint32
	if err := binary.Read(stream, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	// 限制消息大小（最大 100MB）
	if length > 100*1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	// 读取消息内容
	data := make([]byte, length)
	if _, err := io.ReadFull(stream, data); err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	// 反序列化消息
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}


