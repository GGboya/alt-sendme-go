# 使用指南

## 接收方如何接收文件

### 完整流程

#### 步骤 1：发送方启动分享

**发送方**在终端运行：

```bash
./sendme send <文件路径> [--port <端口>]
```

示例：
```bash
./sendme send document.pdf
# 或指定端口
./sendme send document.pdf --port 9000
```

**发送方会看到：**
```
File sharing started!
File: document.pdf
Size: 2.5 MB

Share this ticket with the receiver:

eyJwZWVyX2FkZHIiOiIvaXA0LzEyNy4wLjAuMS90Y3AvOTAwMC9wMnAvUW1OZVhVU3NKbm9hZmNRa1hpZXp6Z0FCM2J5VHN4eEVVcUZoZkxvMlhjVnM4dyIsImZpbGVfaGFzaCI6ImI4ZmFhMjhlMmVjOWJlNmQ4NWMyN2EwNWEyNzkxYTU2MzIxZWY5N2Q3NzJkYTlkZDNlNTBiOTM4YWM4M2RhNzMiLCJmaWxlX25hbWUiOiJkb2N1bWVudC5wZGYiLCJmaWxlX3NpemUiOjI2MjE0NDB9=

Waiting for receiver to connect...
Press Ctrl+C to stop sharing
```

#### 步骤 2：发送方分享 Ticket

发送方需要将显示的 **ticket**（Base64 编码的长字符串）分享给接收方。

分享方式：
- 复制 ticket 字符串
- 通过聊天工具、邮件等方式发送给接收方
- 或者让接收方直接看到发送方的终端输出

#### 步骤 3：接收方接收文件

**接收方**在终端运行：

```bash
./sendme receive <ticket> [--output <输出路径>] [--port <端口>]
```

**基本用法：**
```bash
# 使用 ticket 接收文件（保存到当前目录）
./sendme receive eyJwZWVyX2FkZHIiOiIvaXA0LzEyNy4wLjAuMS90Y3AvOTAwMC9wMnAvUW1OZVhVU3NKbm9hZmNRa1hpZXp6Z0FCM2J5VHN4eEVVcUZoZkxvMlhjVnM4dyIsImZpbGVfaGFzaCI6ImI4ZmFhMjhlMmVjOWJlNmQ4NWMyN2EwNWEyNzkxYTU2MzIxZWY5N2Q3NzJkYTlkZDNlNTBiOTM4YWM4M2RhNzMiLCJmaWxlX25hbWUiOiJkb2N1bWVudC5wZGYiLCJmaWxlX3NpemUiOjI2MjE0NDB9=
```

**指定输出目录：**
```bash
# 保存到指定目录（会自动使用原文件名）
./sendme receive <ticket> --output ./downloads

# 保存到指定路径（可以重命名）
./sendme receive <ticket> --output ./downloads/my_document.pdf
```

**指定端口（如果发送方和接收方在同一台机器）：**
```bash
./sendme receive <ticket> --port 9001 --output ./downloads
```

#### 步骤 4：接收过程

接收方会看到：

```
Receiving file: document.pdf
Size: 2.5 MB

Downloading [████████████████████] 100%  2.5 MB/s
File received successfully: document.pdf (2621440 bytes)
```

### 详细示例

#### 示例 1：局域网传输

**机器 A（发送方）：**
```bash
# 启动发送
./sendme send large_file.zip --port 9000
```

**机器 B（接收方）：**
```bash
# 接收文件到 downloads 目录
./sendme receive <ticket> --port 9001 --output ~/downloads
```

#### 示例 2：同一台机器测试

**终端 1（发送方）：**
```bash
./sendme send test.txt --port 9000
```

**终端 2（接收方）：**
```bash
./sendme receive <ticket> --port 9001 --output received_test.txt
```

### 接收端参数说明

#### `--output` / `-o`：输出路径

- **如果指定的是目录**：文件会保存到该目录，使用原文件名
  ```bash
  ./sendme receive <ticket> --output ./downloads
  # 文件保存为: ./downloads/document.pdf
  ```

- **如果指定的是文件路径**：文件会保存到指定路径（可以重命名）
  ```bash
  ./sendme receive <ticket> --output ./my_file.pdf
  # 文件保存为: ./my_file.pdf
  ```

- **如果不指定**：文件保存到当前目录，使用原文件名
  ```bash
  ./sendme receive <ticket>
  # 文件保存为: ./document.pdf
  ```

#### `--port` / `-p`：监听端口

- 默认值：`0`（随机端口）
- 如果发送方和接收方在同一台机器，建议使用不同端口
- 示例：发送方用 9000，接收方用 9001

### 接收过程中的验证

接收端会自动进行以下验证：

1. **Ticket 验证**：检查 ticket 格式是否正确
2. **连接验证**：确保能连接到发送方节点
3. **文件哈希验证**：接收完成后验证 SHA256 哈希值
4. **文件完整性**：如果哈希不匹配，会自动删除不完整的文件

### 常见问题

#### Q: Ticket 是什么？

A: Ticket 是一个 Base64 编码的 JSON 字符串，包含：
- 发送方的 P2P 节点地址
- 文件的 SHA256 哈希值
- 文件名和大小

#### Q: 接收方需要知道发送方的 IP 地址吗？

A: 不需要。Ticket 中已经包含了连接所需的所有信息。

#### Q: 如果接收失败怎么办？

A: 
- 检查网络连接
- 确认发送方仍在运行
- 检查防火墙设置
- 尝试使用不同的端口

#### Q: 可以同时接收多个文件吗？

A: 当前版本一次只能接收一个文件。需要接收多个文件时，需要多次运行接收命令。

#### Q: 接收的文件会覆盖已存在的文件吗？

A: 不会。如果输出文件已存在，程序会报错并退出，防止意外覆盖。

### 安全提示

1. **Ticket 有效期**：只要发送方程序在运行，ticket 就有效
2. **文件完整性**：接收端会自动验证文件完整性
3. **网络传输**：当前版本在局域网内传输，数据不经过第三方服务器

