package ticket

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Ticket 包含连接到发送方所需的信息
type Ticket struct {
	PeerAddr string `json:"peer_addr"` // libp2p 节点地址
	FileHash string `json:"file_hash"` // 文件哈希
	FileName string `json:"file_name"` // 文件名
	FileSize int64  `json:"file_size"` // 文件大小
}

// Encode 将 Ticket 编码为 Base64 字符串
func (t *Ticket) Encode() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ticket: %w", err)
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

// Decode 从 Base64 字符串解码 Ticket
func Decode(encoded string) (*Ticket, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ticket: %w", err)
	}

	var ticket Ticket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ticket: %w", err)
	}

	if ticket.PeerAddr == "" {
		return nil, fmt.Errorf("invalid ticket: missing peer address")
	}

	if ticket.FileHash == "" {
		return nil, fmt.Errorf("invalid ticket: missing file hash")
	}

	return &ticket, nil
}


