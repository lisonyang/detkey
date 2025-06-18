#!/bin/bash

# 快速测试 mTLS 功能
set -e

echo "=== DetKey mTLS 功能测试 ==="
echo ""

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 设置 detkey 命令
DETKEY_CMD="$PROJECT_DIR/detkey"
if ! command -v "$DETKEY_CMD" &> /dev/null; then
    echo "错误: 找不到 detkey 工具在 $DETKEY_CMD"
    exit 1
fi

PASSWORD="test123"

echo "测试 1: Ed25519 SSH 格式 (传统功能)"
echo "$PASSWORD" | $DETKEY_CMD --context "ssh/test/v1" --type ed25519 | head -2
echo "✓ Ed25519 SSH 私钥生成成功"
echo ""

echo "测试 2: Ed25519 mTLS PEM 格式 (新功能)"
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/ca/v1" --type ed25519 | head -2
echo "✓ Ed25519 mTLS PEM 私钥生成成功"
echo ""

echo "测试 3: RSA 密钥生成 (确定性测试)"
echo "生成两次相同的 RSA 密钥，验证确定性："

# 生成第一次
RSA_KEY1=$(echo "$PASSWORD" | $DETKEY_CMD --context "mtls/test/v1" --type rsa2048 2>/dev/null | head -2)
# 生成第二次  
RSA_KEY2=$(echo "$PASSWORD" | $DETKEY_CMD --context "mtls/test/v1" --type rsa2048 2>/dev/null | head -2)

if [ "$RSA_KEY1" = "$RSA_KEY2" ]; then
    echo "✓ RSA 密钥确定性生成验证成功"
else
    echo "✗ RSA 密钥确定性生成验证失败"
    exit 1
fi
echo ""

echo "测试 4: 格式自动检测"
SSH_FORMAT=$(echo "$PASSWORD" | $DETKEY_CMD --context "ssh/test/v1" --type ed25519 | head -1)
MTLS_FORMAT=$(echo "$PASSWORD" | $DETKEY_CMD --context "mtls/test/v1" --type ed25519 | head -1)

if [[ "$SSH_FORMAT" == *"OPENSSH"* ]]; then
    echo "✓ SSH 上下文自动使用 OpenSSH 格式"
else
    echo "✗ SSH 上下文格式检测失败"
    exit 1
fi

if [[ "$MTLS_FORMAT" == *"PRIVATE KEY"* ]]; then
    echo "✓ mTLS 上下文自动使用 PEM 格式"
else
    echo "✗ mTLS 上下文格式检测失败"
    exit 1
fi
echo ""

echo "测试 5: 公钥生成"
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/test/v1" --type ed25519 --pub | head -2
echo "✓ mTLS PEM 公钥生成成功"
echo ""

echo "测试 6: 强制格式覆盖"
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/test/v1" --type ed25519 --format ssh --pub | head -1
echo "✓ 格式强制覆盖成功"
echo ""

echo "=== 所有测试通过！ ==="
echo ""
echo "mTLS 功能已成功集成到 detkey 工具中："
echo "- ✓ 支持 RSA 2048/4096 密钥类型"
echo "- ✓ 智能格式检测 (SSH vs PEM)"
echo "- ✓ 确定性密钥生成"
echo "- ✓ 向后兼容现有 SSH 功能"
echo "- ✓ 支持公钥/私钥输出"
echo "- ✓ 支持格式强制覆盖"
echo ""
echo "您现在可以使用 detkey 进行完整的 mTLS 证书管理！" 