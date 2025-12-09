#!/bin/bash
set -e

echo "=== 开始文件传输测试 ==="
echo ""

# 清理之前的测试文件
rm -f received_file.txt sender.log receiver.log

# 启动发送端（后台运行）
echo "1. 启动发送端..."
./sendme send test_file.txt --port 9000 > sender.log 2>&1 &
SENDER_PID=$!
echo "   发送端 PID: $SENDER_PID"

# 等待 ticket 生成（增加等待时间）
sleep 4

# 显示发送端输出用于调试
echo "   发送端输出:"
cat sender.log | head -15
echo ""

# 从日志中提取 ticket（查找空行后的 Base64 字符串）
TICKET=$(grep -A 2 "Share this ticket" sender.log | grep -v "Share this ticket" | grep -v "^$" | head -1 | tr -d '[:space:]')

if [ -z "$TICKET" ]; then
    echo "   ❌ 无法获取 ticket，尝试其他方法..."
    # 尝试直接查找 Base64 格式的字符串
    TICKET=$(grep -E "^[A-Za-z0-9_-]{50,}" sender.log | head -1 | tr -d '[:space:]')
fi

if [ -z "$TICKET" ]; then
    echo "   ❌ 仍然无法获取 ticket"
    echo "   完整日志:"
    cat sender.log
    kill $SENDER_PID 2>/dev/null
    exit 1
fi

echo "   ✓ Ticket 已生成"
echo "   Ticket (前50字符): ${TICKET:0:50}..."
echo ""

# 启动接收端
echo "2. 启动接收端..."
./sendme receive "$TICKET" --port 9001 --output received_file.txt > receiver.log 2>&1
RECEIVER_EXIT=$?

# 等待一下确保传输完成
sleep 2

# 停止发送端
echo "3. 停止发送端..."
kill $SENDER_PID 2>/dev/null || true
wait $SENDER_PID 2>/dev/null || true

echo ""

# 验证结果
echo "4. 验证传输结果..."

if [ $RECEIVER_EXIT -ne 0 ]; then
    echo "   ❌ 接收端失败 (退出码: $RECEIVER_EXIT)"
    echo "   错误信息:"
    cat receiver.log
    exit 1
fi

if [ ! -f received_file.txt ]; then
    echo "   ❌ 接收文件不存在"
    echo "   接收端日志:"
    cat receiver.log
    exit 1
fi

# 比较文件
if diff -q test_file.txt received_file.txt > /dev/null; then
    echo "   ✓ 文件内容匹配"
else
    echo "   ❌ 文件内容不匹配"
    echo "   原始文件:"
    cat test_file.txt
    echo "   接收文件:"
    cat received_file.txt
    exit 1
fi

# 比较哈希
ORIGINAL_HASH=$(sha256sum test_file.txt | cut -d' ' -f1)
RECEIVED_HASH=$(sha256sum received_file.txt | cut -d' ' -f1)

if [ "$ORIGINAL_HASH" = "$RECEIVED_HASH" ]; then
    echo "   ✓ 文件哈希匹配: $ORIGINAL_HASH"
else
    echo "   ❌ 文件哈希不匹配"
    echo "   原始: $ORIGINAL_HASH"
    echo "   接收: $RECEIVED_HASH"
    exit 1
fi

# 比较大小
ORIGINAL_SIZE=$(stat -f%z test_file.txt 2>/dev/null || stat -c%s test_file.txt 2>/dev/null)
RECEIVED_SIZE=$(stat -f%z received_file.txt 2>/dev/null || stat -c%s received_file.txt 2>/dev/null)

if [ "$ORIGINAL_SIZE" = "$RECEIVED_SIZE" ]; then
    echo "   ✓ 文件大小匹配: $ORIGINAL_SIZE 字节"
else
    echo "   ❌ 文件大小不匹配"
    echo "   原始: $ORIGINAL_SIZE 字节"
    echo "   接收: $RECEIVED_SIZE 字节"
    exit 1
fi

echo ""
echo "=== ✅ 测试通过！ ==="
echo ""
echo "测试文件:"
echo "  发送: test_file.txt ($ORIGINAL_SIZE 字节)"
echo "  接收: received_file.txt ($RECEIVED_SIZE 字节)"
echo "  哈希: $ORIGINAL_HASH"
