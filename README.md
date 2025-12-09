# AltSendme Go

一个使用 Go 和 libp2p 构建的简单 P2P 文件传输工具。

## 功能特性

- ✅ 单文件传输
- ✅ P2P 直连传输
- ✅ 文件完整性校验（SHA256）
- ✅ 传输进度显示
- ✅ Ticket 系统（Base64 编码）
- ✅ 基本错误处理

## 安装

```bash
go build -o sendme ./cmd/sendme
```

## 使用方法

### 发送文件

```bash
./sendme send <文件路径> [--port <端口>]
```

示例：
```bash
./sendme send document.pdf
./sendme send document.pdf --port 9000
```

发送方会生成一个 ticket，将其分享给接收方。

### 接收文件

```bash
./sendme receive <ticket> [--output <输出路径>] [--port <端口>]
```

示例：
```bash
./sendme receive <ticket>
./sendme receive <ticket> --output ./downloads
./sendme receive <ticket> --output ./downloads/file.pdf
```

## 项目结构

```
alt-sendme-go/
├── cmd/sendme/          # CLI 主程序
├── internal/
│   ├── sender/         # 发送端逻辑
│   ├── receiver/       # 接收端逻辑
│   ├── p2p/            # P2P 网络封装
│   ├── ticket/         # Ticket 系统
│   └── utils/          # 工具函数
└── README.md
```

## 技术栈

- **Go 1.21+**
- **libp2p**: P2P 网络库
- **cobra**: CLI 框架
- **progressbar**: 进度条显示

## 限制

第一个版本的限制：

- ❌ 仅支持单文件传输（不支持目录）
- ❌ 仅支持局域网传输（无 NAT 穿透）
- ❌ 无加密传输（后续版本添加）

## 开发

```bash
# 安装依赖
go mod tidy

# 构建
go build ./cmd/sendme

# 运行
./sendme send test.txt
```

## 许可证

AGPL-3.0


