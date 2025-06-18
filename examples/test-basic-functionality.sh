#!/bin/bash

# DetKey 基本功能测试脚本
# 此脚本演示 detkey 的确定性密钥生成功能

echo "=== DetKey 基本功能测试 ==="
echo

# 确保 detkey 可执行文件存在
if [ ! -f "../detkey" ]; then
    echo "错误: 未找到 detkey 可执行文件。请先运行 'make build' 构建项目。"
    exit 1
fi

DETKEY="../detkey"
TEST_PASSWORD="test-password-123"
TEST_CONTEXT="ssh/test-server/v1"

echo "测试参数:"
echo "- 密码: $TEST_PASSWORD"
echo "- 上下文: $TEST_CONTEXT"
echo

echo "=== 测试 1: 确定性生成验证 ==="
echo "生成相同上下文的密钥两次，验证结果一致..."

# 生成第一次
echo "第一次生成公钥:"
PUBKEY1=$(echo "$TEST_PASSWORD" | $DETKEY --context "$TEST_CONTEXT" --pub)
echo "$PUBKEY1"
echo

# 生成第二次
echo "第二次生成公钥:"
PUBKEY2=$(echo "$TEST_PASSWORD" | $DETKEY --context "$TEST_CONTEXT" --pub)
echo "$PUBKEY2"
echo

# 比较结果
if [ "$PUBKEY1" = "$PUBKEY2" ]; then
    echo "✅ 确定性测试通过: 相同输入产生相同输出"
else
    echo "❌ 确定性测试失败: 相同输入产生不同输出"
    exit 1
fi
echo

echo "=== 测试 2: 不同上下文生成不同密钥 ==="
CONTEXT1="ssh/server-1/v1"
CONTEXT2="ssh/server-2/v1"

echo "上下文 1: $CONTEXT1"
PUBKEY_CTX1=$(echo "$TEST_PASSWORD" | $DETKEY --context "$CONTEXT1" --pub)
echo "$PUBKEY_CTX1"
echo

echo "上下文 2: $CONTEXT2"
PUBKEY_CTX2=$(echo "$TEST_PASSWORD" | $DETKEY --context "$CONTEXT2" --pub)
echo "$PUBKEY_CTX2"
echo

if [ "$PUBKEY_CTX1" != "$PUBKEY_CTX2" ]; then
    echo "✅ 上下文隔离测试通过: 不同上下文产生不同密钥"
else
    echo "❌ 上下文隔离测试失败: 不同上下文产生相同密钥"
    exit 1
fi
echo

echo "=== 测试 3: 私钥和公钥格式验证 ==="

echo "生成私钥 (PEM 格式):"
PRIVATE_KEY=$(echo "$TEST_PASSWORD" | $DETKEY --context "$TEST_CONTEXT")
echo "$PRIVATE_KEY" | head -5
echo "... (截断显示)"
echo "$PRIVATE_KEY" | tail -5
echo

# 验证私钥格式
if echo "$PRIVATE_KEY" | grep -q "BEGIN OPENSSH PRIVATE KEY"; then
    echo "✅ 私钥格式验证通过: 标准 OpenSSH PEM 格式"
else
    echo "❌ 私钥格式验证失败: 非标准格式"
    exit 1
fi

# 验证公钥格式
if echo "$PUBKEY1" | grep -q "^ssh-ed25519 "; then
    echo "✅ 公钥格式验证通过: 标准 SSH 公钥格式"
else
    echo "❌ 公钥格式验证失败: 非标准格式"
    exit 1
fi
echo

echo "=== 测试 4: 版本控制功能 ==="
VERSION_V1="ssh/test-server/v1"
VERSION_V2="ssh/test-server/v2"

echo "版本 v1:"
PUBKEY_V1=$(echo "$TEST_PASSWORD" | $DETKEY --context "$VERSION_V1" --pub)
echo "$PUBKEY_V1"
echo

echo "版本 v2:"
PUBKEY_V2=$(echo "$TEST_PASSWORD" | $DETKEY --context "$VERSION_V2" --pub)
echo "$PUBKEY_V2"
echo

if [ "$PUBKEY_V1" != "$PUBKEY_V2" ]; then
    echo "✅ 版本控制测试通过: 不同版本产生不同密钥"
else
    echo "❌ 版本控制测试失败: 不同版本产生相同密钥"
    exit 1
fi
echo

echo "=== 全部测试通过! ==="
echo "DetKey 工具工作正常，可以安全使用。"
echo
echo "接下来您可以:"
echo "1. 运行 'make install' 将工具安装到系统路径"
echo "2. 开始使用真实的主密码和上下文"
echo "3. 为常用服务器创建 shell 别名" 