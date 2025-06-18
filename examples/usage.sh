#!/bin/bash

# DetKey 使用示例脚本
# 此脚本演示了 detkey 工具的各种使用方法

echo "=== DetKey 使用示例 ==="
echo

# 确保 detkey 可执行文件存在
if [ ! -f "../detkey" ]; then
    echo "错误: 未找到 detkey 可执行文件。请先运行 'make build' 构建项目。"
    exit 1
fi

DETKEY="../detkey"

echo "1. 生成私钥示例:"
echo "命令: $DETKEY --context 'ssh/example-server/v1'"
echo "注意: 运行时会提示输入主密码"
echo

echo "2. 生成公钥示例:"
echo "命令: $DETKEY --context 'ssh/example-server/v1' --pub"
echo

echo "3. 实际场景 - 部署公钥到服务器:"
echo "命令: $DETKEY --context 'ssh/prod-server/v1' --pub | ssh user@server 'cat >> ~/.ssh/authorized_keys'"
echo

echo "4. 实际场景 - 使用私钥登录服务器:"
echo "命令: ssh -i <($DETKEY --context 'ssh/prod-server/v1') user@server"
echo

echo "5. 不同上下文生成不同密钥:"
echo "- ssh/production/web-server/v1"
echo "- ssh/staging/database/v1" 
echo "- ssh/personal/vps/v1"
echo "- git/github/personal/v1"
echo

echo "6. 创建便捷别名 (添加到 ~/.bashrc 或 ~/.zshrc):"
cat << 'EOF'
alias ssh-prod='ssh -i <(detkey --context "ssh/production/web-server/v1") user@prod-server'
alias ssh-staging='ssh -i <(detkey --context "ssh/staging/database/v1") user@staging-server'
alias ssh-personal='ssh -i <(detkey --context "ssh/personal/vps/v1") user@personal-server'
EOF
echo

echo "=== 安全提示 ==="
echo "1. 使用强主密码，包含大小写字母、数字和特殊字符"
echo "2. 不要在不信任的环境中使用此工具"
echo "3. 定期轮换重要服务的密钥（通过更改上下文版本号）"
echo "4. 私钥永远不会保存到磁盘，仅在内存中生成和使用"
echo

echo "运行完整示例请使用: ./test-basic-functionality.sh" 