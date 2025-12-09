# AltSendme Go

一个使用 Go 和 libp2p 构建的简单 P2P 文件传输工具。

## 功能特性

- ✅ 单文件传输
- ✅ P2P 直连传输
- ✅ **跨网络传输**（支持 NAT 穿透和中继服务器）
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

**完整流程：**

1. **发送方启动分享**，获得 ticket：
   ```bash
   ./sendme send document.pdf
   # 会显示一个 ticket，类似：
   # eyJwZWVyX2FkZHIiOiIvaXA0LzEyNy4wLjAuMS90Y3AvOTAwMC9wMnAv...
   ```

2. **发送方将 ticket 分享给接收方**（复制 ticket 字符串）

3. **接收方使用 ticket 接收文件**：
   ```bash
   # 基本用法（保存到当前目录）
   ./sendme receive <ticket>
   
   # 保存到指定目录
   ./sendme receive <ticket> --output ./downloads
   
   # 保存到指定路径（可重命名）
   ./sendme receive <ticket> --output ./downloads/my_file.pdf
   
   # 指定端口（同一台机器时使用不同端口）
   ./sendme receive <ticket> --port 9001 --output ./downloads
   ```

**接收过程：**
- 程序会自动连接到发送方
- 显示传输进度条
- 自动验证文件完整性（SHA256）
- 完成后显示文件保存路径

**详细使用说明请查看 [USAGE.md](USAGE.md)**

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

## 网络传输说明

### 跨网络传输

AltSendme Go 支持跨网络文件传输，使用 libp2p 的 AutoRelay 和 AutoNAT 功能：

- **自动 NAT 穿透**：优先尝试直连，如果直连失败会自动使用中继服务器
- **自动中继发现**：自动发现和使用网络中的中继服务器
- **智能连接**：连接时会自动选择最佳路径（直连或中继）

### 工作原理

1. **直连优先**：首先尝试直接连接到对方
2. **自动回退**：如果直连失败（例如在 NAT 后面），自动使用中继服务器
3. **透明传输**：整个过程对用户透明，无需手动配置

首次连接可能需要几秒钟来发现和连接中继服务器。

## 限制

第一个版本的限制：

- ❌ 仅支持单文件传输（不支持目录）
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


