#!/bin/bash
# 快速测试脚本

echo "=== AltSendme Go 快速测试 ==="
echo ""

# 创建测试文件
echo "1. 创建测试文件..."
echo "Hello from AltSendme Go! $(date)" > test_file.txt
echo "   文件已创建: test_file.txt"
echo ""

# 构建程序
echo "2. 构建程序..."
go build -o sendme ./cmd/sendme
if [ $? -ne 0 ]; then
    echo "   构建失败！"
    exit 1
fi
echo "   构建成功！"
echo ""

# 显示使用说明
echo "3. 测试步骤："
echo ""
echo "   终端 1 (发送端):"
echo "   $ ./sendme send test_file.txt"
echo ""
echo "   终端 2 (接收端):"
echo "   $ ./sendme receive <ticket> --output received_file.txt"
echo ""
echo "   验证:"
echo "   $ diff test_file.txt received_file.txt"
echo "   $ sha256sum test_file.txt received_file.txt"
echo ""
echo "=== 准备就绪 ==="
