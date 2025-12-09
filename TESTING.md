# 测试指南

## 准备工作

1. **构建程序**
```bash
cd /home/syh/study/github/alt-sendme-go
go build -o sendme ./cmd/sendme
```

2. **准备测试文件**
```bash
# 创建一个测试文件
echo "Hello, this is a test file for P2P transfer!" > test.txt

# 或者创建一个稍大一点的文件（用于测试进度显示）
dd if=/dev/urandom of=test_large.bin bs=1M count=10 2>/dev/null
```

## 测试方法

### 方法一：两个终端窗口测试（推荐）

#### 终端 1 - 发送端
```bash
cd /home/syh/study/github/alt-sendme-go
./sendme send test.txt
```

发送端会：
1. 计算文件哈希
2. 创建 P2P 节点
3. 显示生成的 ticket（类似：`eyJwZWVyX2FkZHI...`）
4. 等待接收方连接

**重要**：复制显示的 ticket，稍后在接收端使用。

#### 终端 2 - 接收端
```bash
cd /home/syh/study/github/alt-sendme-go
./sendme receive <粘贴ticket> --output ./received_test.txt
```

接收端会：
1. 解析 ticket
2. 连接到发送方
3. 下载文件
4. 验证文件完整性
5. 显示传输进度

### 方法二：使用不同端口（同一台机器）

如果两个终端在同一台机器上，可以使用不同端口：

#### 终端 1 - 发送端（端口 9000）
```bash
./sendme send test.txt --port 9000
```

#### 终端 2 - 接收端（端口 9001）
```bash
./sendme receive <ticket> --port 9001 --output ./received_test.txt
```

### 方法三：两台机器测试（局域网）

#### 机器 A - 发送端
```bash
./sendme send test.txt --port 9000
```

#### 机器 B - 接收端
```bash
./sendme receive <ticket> --port 9001 --output ./received_test.txt
```

确保两台机器在同一局域网内。

## 验证传输成功

### 1. 检查文件是否存在
```bash
ls -lh received_test.txt
```

### 2. 比较文件哈希
```bash
# 发送端文件哈希
sha256sum test.txt

# 接收端文件哈希
sha256sum received_test.txt
```

两个哈希值应该完全一致。

### 3. 比较文件内容
```bash
diff test.txt received_test.txt
```

如果没有输出，说明文件内容完全一致。

### 4. 检查文件大小
```bash
# 发送端
ls -lh test.txt

# 接收端
ls -lh received_test.txt
```

文件大小应该完全一致。

## 测试场景

### 场景 1：小文件测试（< 1MB）
```bash
# 发送端
./sendme send test.txt

# 接收端（使用显示的 ticket）
./sendme receive <ticket> --output ./received_test.txt
```

### 场景 2：大文件测试（> 10MB）
```bash
# 创建大文件
dd if=/dev/urandom of=large_file.bin bs=1M count=50 2>/dev/null

# 发送端
./sendme send large_file.bin

# 接收端
./sendme receive <ticket> --output ./received_large_file.bin
```

观察进度条是否正常显示。

### 场景 3：错误处理测试

#### 测试无效 ticket
```bash
./sendme receive invalid_ticket_here
```
应该显示错误信息。

#### 测试文件不存在
```bash
./sendme send nonexistent_file.txt
```
应该显示文件不存在的错误。

#### 测试输出文件已存在
```bash
# 先创建一个文件
touch output.txt

# 尝试接收并覆盖
./sendme receive <ticket> --output ./output.txt
```
应该显示文件已存在的错误。

## 调试技巧

### 1. 查看详细输出
程序会输出：
- 文件信息（名称、大小）
- Ticket（Base64 编码）
- 连接状态
- 传输进度
- 完成信息

### 2. 检查网络连接
```bash
# 查看监听端口
netstat -tuln | grep 9000

# 或使用 ss
ss -tuln | grep 9000
```

### 3. 查看进程
```bash
# 查看 sendme 进程
ps aux | grep sendme
```

## 常见问题

### 问题 1：连接失败
**原因**：防火墙阻止或端口被占用
**解决**：
- 检查防火墙设置
- 使用不同的端口（`--port` 参数）
- 确保两台机器在同一网络

### 问题 2：Ticket 无效
**原因**：Ticket 格式错误或已过期
**解决**：
- 确保完整复制 ticket（包括所有字符）
- 使用最新生成的 ticket

### 问题 3：文件哈希不匹配
**原因**：传输过程中数据损坏
**解决**：
- 检查网络连接稳定性
- 重新传输文件

## 自动化测试脚本

创建一个简单的测试脚本：

```bash
#!/bin/bash
# test.sh

echo "Creating test file..."
echo "Test content $(date)" > test.txt

echo "Starting sender in background..."
./sendme send test.txt --port 9000 > sender.log 2>&1 &
SENDER_PID=$!

# 等待 ticket 生成
sleep 2

# 从日志中提取 ticket（需要根据实际输出调整）
TICKET=$(grep -oP '(?<=ticket: ).*' sender.log | head -1)

if [ -z "$TICKET" ]; then
    echo "Failed to get ticket from sender"
    kill $SENDER_PID
    exit 1
fi

echo "Ticket: $TICKET"
echo "Starting receiver..."
./sendme receive "$TICKET" --port 9001 --output received_test.txt

# 验证文件
if diff test.txt received_test.txt; then
    echo "✓ Test passed: Files match!"
else
    echo "✗ Test failed: Files differ"
fi

# 清理
kill $SENDER_PID
rm -f test.txt received_test.txt sender.log
```

使用方法：
```bash
chmod +x test.sh
./test.sh
```


