#!/bin/bash

# mTLS 演示脚本
# 演示如何使用 detkey 工具生成 mTLS 所需的所有证书和密钥

set -e

echo "=== DetKey mTLS 演示 ==="
echo "此脚本演示如何使用 detkey 工具生成 mTLS 环境"
echo ""

# 检查 detkey 是否存在
if ! command -v ./detkey &> /dev/null && ! command -v detkey &> /dev/null; then
    echo "错误: 找不到 detkey 工具。请先构建或安装 detkey。"
    exit 1
fi

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 设置 detkey 命令
DETKEY_CMD="$PROJECT_DIR/detkey"
if ! command -v "$DETKEY_CMD" &> /dev/null; then
    if command -v ./detkey &> /dev/null; then
        DETKEY_CMD="./detkey"
    elif command -v detkey &> /dev/null; then
        DETKEY_CMD="detkey"
    else
        echo "错误: 找不到 detkey 工具。请先构建或安装 detkey。"
        exit 1
    fi
fi

# 设置密码（从标准输入读取或使用默认密码）
if [ -t 0 ]; then
    # 如果是终端，使用默认密码进行演示
    PASSWORD="demo123"
    echo "使用默认演示密码"
else
    # 如果是管道输入，读取密码
    read -r PASSWORD
    echo "使用输入的密码"
fi

# 创建临时目录
TEMP_DIR=$(mktemp -d)
echo "创建临时目录: $TEMP_DIR"
cd "$TEMP_DIR"

echo ""
echo "=== 步骤 1: 生成 CA 私钥和证书 ==="

# 生成 CA 私钥 (RSA 2048位，演示用)
echo "生成 CA 私钥..."
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/ca/v1" --type rsa2048 > ca.key
echo "✓ CA 私钥已生成 (ca.key)"

# 创建自签名的 CA 证书
echo "创建 CA 证书..."
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=DetKey Demo/OU=CA/CN=DetKey Demo CA"
echo "✓ CA 证书已创建 (ca.crt)"

echo ""
echo "=== 步骤 2: 生成服务器私钥和证书 ==="

# 生成服务器私钥
echo "生成服务器私钥..."
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/server/api.example.com/v1" --type rsa2048 > server.key
echo "✓ 服务器私钥已生成 (server.key)"

# 创建服务器证书签名请求 (CSR)
echo "创建服务器 CSR..."
openssl req -new -key server.key -out server.csr \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=DetKey Demo/OU=Server/CN=api.example.com"
echo "✓ 服务器 CSR 已创建 (server.csr)"

# 用 CA 签发服务器证书
echo "签发服务器证书..."
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt -days 365 -sha256
echo "✓ 服务器证书已签发 (server.crt)"

echo ""
echo "=== 步骤 3: 生成客户端私钥和证书 ==="

# 生成客户端私钥
echo "生成客户端私钥..."
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/client/dashboard/v1" --type rsa2048 > client.key
echo "✓ 客户端私钥已生成 (client.key)"

# 创建客户端证书签名请求 (CSR)
echo "创建客户端 CSR..."
openssl req -new -key client.key -out client.csr \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=DetKey Demo/OU=Client/CN=dashboard-client"
echo "✓ 客户端 CSR 已创建 (client.csr)"

# 用 CA 签发客户端证书
echo "签发客户端证书..."
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt -days 365 -sha256
echo "✓ 客户端证书已签发 (client.crt)"

echo ""
echo "=== 步骤 4: 验证证书 ==="

echo "验证 CA 证书..."
openssl x509 -in ca.crt -text -noout | grep -A 2 "Subject:"

echo ""
echo "验证服务器证书..."
openssl verify -CAfile ca.crt server.crt
openssl x509 -in server.crt -text -noout | grep -A 2 "Subject:"

echo ""
echo "验证客户端证书..."
openssl verify -CAfile ca.crt client.crt
openssl x509 -in client.crt -text -noout | grep -A 2 "Subject:"

echo ""
echo "=== 生成的文件列表 ==="
ls -la *.crt *.key

echo ""
echo "=== mTLS 设置完成！ ==="
echo ""
echo "生成的文件位置: $TEMP_DIR"
echo ""
echo "文件说明:"
echo "  ca.crt + ca.key     - CA 证书和私钥"
echo "  server.crt + server.key - 服务器证书和私钥"
echo "  client.crt + client.key - 客户端证书和私钥"
echo ""

echo "=== 重要提示 ==="
echo "1. 所有私钥都可以通过 detkey 和您的主密码重新生成"
echo "2. 只需要备份证书文件 (*.crt)，私钥可以随时重新生成"
echo "3. 如果需要轮换密钥，只需更改上下文版本号 (如 v1 -> v2)"
echo ""

echo "=== 使用示例 ==="
echo "测试 mTLS 连接 (需要支持 mTLS 的服务器):"
echo "curl --cert $TEMP_DIR/client.crt --key $TEMP_DIR/client.key --cacert $TEMP_DIR/ca.crt https://api.example.com"
echo ""

echo "=== DetKey mTLS 命令参考 ==="
echo "重新生成 CA 私钥:      $DETKEY_CMD --context \"mtls/ca/v1\" --type rsa2048"
echo "重新生成服务器私钥:    $DETKEY_CMD --context \"mtls/server/api.example.com/v1\" --type rsa2048"
echo "重新生成客户端私钥:    $DETKEY_CMD --context \"mtls/client/dashboard/v1\" --type rsa2048"
echo ""

echo "演示完成。临时文件保存在: $TEMP_DIR"
echo "如需清理，请运行: rm -rf $TEMP_DIR" 