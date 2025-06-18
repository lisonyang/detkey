# DetKey - 确定性SSH密钥生成器

[English](README.md) | [中文](README_zh.md)

DetKey 是一个强大的命令行工具，允许您使用主密码和上下文字符串来确定性地生成SSH密钥和mTLS证书。这意味着相同的输入总是会产生相同的密钥对，让您能够在任何地方重新生成相同的密钥，而无需存储或传输密钥文件。

## 核心特性

- **确定性生成**: 相同的主密码和上下文总是生成相同的密钥对
- **多种密钥类型**: 支持Ed25519、RSA 2048位和RSA 4096位密钥
- **mTLS支持**: 为双向TLS认证场景生成RSA密钥
- **零依赖**: 编译为单一可执行文件，无需任何外部依赖
- **跨平台**: 支持Linux、macOS、Windows
- **安全优先设计**: 使用Argon2id进行密钥延伸，HKDF进行密钥衍生
- **多种输出格式**: 标准OpenSSH格式和PEM格式
- **智能格式检测**: 根据上下文自动选择合适的格式

## 安装

### 一键安装（推荐）

您可以使用以下命令自动下载并安装最新版本：

```bash
curl -sfL https://raw.githubusercontent.com/lisonyang/detkey/main/install.sh | sh
```

该脚本会自动：
- 检测您的操作系统和CPU架构
- 从GitHub Releases下载对应的二进制文件
- 安装到`/usr/local/bin`目录（可能需要sudo权限）

### 手动安装

1. 访问[Releases页面](https://github.com/lisonyang/detkey/releases)
2. 下载适合您系统的压缩包
3. 解压并将`detkey`可执行文件移动到PATH中的目录

## 使用方法

### 命令行选项

```bash
detkey [选项]

选项:
  --context string    密钥衍生的上下文字符串（必需）
                     示例: 'ssh/server-a/v1', 'mtls/ca/v1'
  --type string      要生成的密钥类型（默认 "ed25519"）
                     选项: ed25519, rsa2048, rsa4096
  --format string    输出格式（默认 "auto"）
                     选项: auto, ssh, pem
  --pub             输出公钥而非私钥
```

### SSH密钥生成

#### 基本SSH用法

```bash
# 生成Ed25519私钥（默认）
./detkey --context "ssh/server-a/v1"

# 生成Ed25519公钥
./detkey --context "ssh/server-a/v1" --pub

# 生成RSA私钥
./detkey --context "ssh/server-b/v1" --type rsa2048
```

#### 实际SSH使用场景

**⚠️ 重要提示：可靠的SSH密钥部署方法**

常见的单行管道命令可能会导致密码输入冲突，因为`detkey`和`ssh`两个程序会同时尝试从终端读取密码。为了确保可靠的部署，请使用下面的**三步文件法**：

**第1步：生成公钥到临时文件**
```bash
# 生成公钥并保存到临时文件
./detkey --context "ssh/prod-server/v1" --pub > /tmp/prod-server.pub
```

**第2步：使用ssh-copy-id部署（推荐）**
```bash
# 使用OpenSSH官方推荐的部署工具
ssh-copy-id -i /tmp/prod-server.pub user@server
```

**第3步：清理临时文件**
```bash
# 删除临时文件
rm /tmp/prod-server.pub
```

**备选部署方法（如果没有ssh-copy-id）：**
```bash
# 第1步：生成公钥到临时文件
./detkey --context "ssh/prod-server/v1" --pub > /tmp/prod-server.pub

# 第2步：手动部署
cat /tmp/prod-server.pub | ssh user@server "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && chmod 700 ~/.ssh"

# 第3步：清理临时文件
rm /tmp/prod-server.pub
```

**部署后，使用可靠的登录方法：**

### SSH登录工作流程

**⚠️ 重要提示：可靠的SSH登录方法**

常见的单行命令如`ssh -i <(./detkey ...)`可能会导致终端控制冲突，其中`detkey`和`ssh`同时尝试与终端交互。这会导致密码输入被干扰。

**为了完全可靠的SSH登录，请使用以下shell函数方法：**

#### 第1步：添加SSH辅助函数

将以下函数添加到您的shell配置文件（`~/.bashrc`、`~/.zshrc`等）：

```bash
#
# detkey_ssh - 安全且可靠的SSH登录函数
#
# 此函数使用临时私钥文件来解决终端控制冲突，
# 确保在任何环境中都能稳定运行。
#
detkey_ssh() {
    # 检查参数
    if [ "$#" -lt 2 ]; then
        echo "用法: detkey_ssh <上下文> <user@host> [其他ssh选项...]"
        return 1
    fi

    local context="$1"
    shift # 移除第一个参数（上下文），其余为ssh命令参数

    # 为私钥创建安全的临时文件
    # mktemp创建一个只有当前用户可读写的文件
    local tmp_key_file
    tmp_key_file=$(mktemp)
    if [ -z "$tmp_key_file" ]; then
        echo "错误：无法创建临时文件。" >&2
        return 1
    fi

    # 设置陷阱确保临时文件被自动删除
    # 无论函数如何退出（成功、失败、中断）。
    # 这是关键的安全措施！
    trap 'rm -f "$tmp_key_file"' EXIT INT TERM

    # --- 第1步：独立生成私钥并写入临时文件 ---
    # 此过程独立运行，没有程序竞争终端访问。
    # 我们将detkey的输出重定向到临时文件。
    if ! detkey --context "$context" > "$tmp_key_file"; then
        echo "错误：detkey生成私钥失败。" >&2
        # 陷阱将在此处触发并自动删除文件
        return 1
    fi
    # 此时会提示您输入主密码 - 请在此处输入。

    # --- 第2步：使用生成的临时私钥文件进行SSH登录 ---
    # 现在ssh从静态文件读取私钥，它将正确获得终端控制权
    # 而不会与任何其他进程冲突。
    echo "正在使用衍生密钥连接..." >&2
    ssh -i "$tmp_key_file" "$@"
    
    # ssh命令完成后，函数退出，陷阱自动删除临时文件。
}
```

**注意**：如果detkey不在您的PATH中，请将`detkey`替换为detkey可执行文件的实际路径。

#### 第2步：重新加载Shell配置

```bash
source ~/.bashrc  # 或 ~/.zshrc
```

#### 第3步：使用可靠的SSH登录

现在您的登录工作流程变成：

```bash
# 连接到生产服务器
detkey_ssh "ssh/prod-server/v1" user@server

# 使用额外的SSH选项连接
detkey_ssh "ssh/prod-server/v1" user@server -p 2222

# 使用端口转发连接
detkey_ssh "ssh/prod-server/v1" user@server -L 8080:localhost:80
```

**为什么此方法是可行的：**
1. **完全时间分离**：`detkey`的密码输入过程在`ssh`开始之前完全完成
2. **静态文件访问**：`ssh`从现有的静态文件读取——这是其最标准和可靠的操作模式
3. **自动安全清理**：`trap`命令确保无论发生什么，私钥文件都会被自动且可靠地销毁

**您的登录体验：**
1. 您运行：`detkey_ssh "ssh/prod-server/v1" user@server`
2. 您看到：`请输入您的主密码：`（输入您的**主密码**）
3. 您看到：`正在使用衍生密钥连接...`
4. 您已登录：`user@server:~$`

不再有密码冲突，不再有混乱——只有可靠、安全的SSH访问。

### mTLS证书生成

DetKey可以为双向TLS（mTLS）认证生成RSA私钥，非常适合微服务、API认证和安全的服务间通信。

#### 快速mTLS设置

生成完整mTLS设置所需的所有私钥：

```bash
# 生成CA私钥
./detkey --context "mtls/ca/v1" --type rsa4096 > ca.key

# 生成服务器私钥
./detkey --context "mtls/server/api.example.com/v1" --type rsa4096 > server.key

# 生成客户端私钥
./detkey --context "mtls/client/dashboard/v1" --type rsa4096 > client.key
```

#### 完整的mTLS证书链

生成私钥后，使用OpenSSL创建证书链：

```bash
# 1. 创建自签名CA证书
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt \
    -subj "/CN=My Internal CA"

# 2. 创建服务器证书
openssl req -new -key server.key -out server.csr \
    -subj "/CN=api.example.com"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt -days 365 -sha256

# 3. 创建客户端证书
openssl req -new -key client.key -out client.csr \
    -subj "/CN=dashboard-client"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt -days 365 -sha256
```

#### mTLS演示脚本

完整的mTLS设置示例请参考`./examples/mtls-demo.sh`。

### 输出格式

DetKey支持多种输出格式并可以自动检测适当的格式：

- **SSH格式**：标准OpenSSH私钥/公钥格式
- **PEM格式**：用于TLS/SSL的PKCS#1（RSA）或PKCS#8（Ed25519）格式

格式根据上下文自动选择：
- `ssh/*`上下文 → SSH格式
- `mtls/*`上下文 → PEM格式
- RSA密钥 → PEM格式（当上下文不明确时）
- Ed25519密钥 → SSH格式（当上下文不明确时）

## 上下文字符串设计

上下文字符串用于区分不同用途并确保密钥隔离。我们建议使用分层命名：

### SSH上下文
```
ssh/production/web-server-1/v1
ssh/staging/database/v1
ssh/personal/vps/v2
git/github/personal/v1
git/gitlab/work/v1
```

### mTLS上下文
```
mtls/ca/v1                           # 证书颁发机构
mtls/server/api.example.com/v1       # 服务器证书
mtls/server/internal.service/v1      # 内部服务
mtls/client/dashboard/v1             # 客户端证书
mtls/client/monitoring/v1            # 监控客户端
```

### 版本控制

需要密钥轮换时更改版本号：

```
mtls/ca/v1    → mtls/ca/v2           # CA密钥轮换
ssh/prod/v1   → ssh/prod/v2          # SSH密钥轮换
```

## 密钥类型和使用场景

| 密钥类型 | 位数 | 使用场景 | 性能 |
|----------|------|----------|------|
| `ed25519` | 256 | SSH认证、Git签名 | 最快 |
| `rsa2048` | 2048 | 传统TLS、旧系统 | 中等 |
| `rsa4096` | 4096 | 高安全性TLS、CA密钥 | 较慢 |

**建议：**
- **SSH**：使用`ed25519`（默认）以获得最佳性能和安全性
- **mTLS**：CA密钥使用`rsa4096`，客户端/服务器密钥使用`rsa2048`
- **传统系统**：不支持`ed25519`时使用`rsa2048`

## 安全考虑

### 优势

- **密钥延伸**：使用Argon2id算法使暴力破解成本极高
- **隔离性**：不同上下文生成完全独立的密钥
- **不存储**：密钥在内存中生成，使用后立即销毁
- **确定性**：无需担心密钥丢失或备份
- **完美前向保密**：通过更改版本号可以轮换密钥

### 权衡

- **主密码强度**：工具的安全性依赖于您主密码的强度
- **离线攻击**：如果攻击者获得工具和已知的密钥对，可能尝试暴力破解主密码

### 最佳实践

1. **使用强主密码**：建议使用包含大小写字母、数字和特殊字符的长密码
2. **保护工具安全**：不要在不信任的环境中使用
3. **上下文版本控制**：需要轮换密钥时更改上下文中的版本号
4. **定期轮换**：定期轮换重要服务的密钥
5. **备份证书**：虽然私钥是确定性的，但要备份您的证书（.crt文件）

## DetKey的mTLS优势

### 传统mTLS痛点
- **密钥管理**：安全地存储和分发私钥
- **备份与恢复**：丢失私钥的风险
- **密钥轮换**：在各处替换密钥的复杂过程

### DetKey解决方案
- **确定性生成**：仅使用主密码即可在任何地方重新生成相同的密钥
- **无密钥存储**：私钥无需存储在磁盘上
- **轻松轮换**：更改上下文字符串中的版本即可轮换密钥
- **简化分发**：只需分发证书，无需分发私钥

## 故障排除

### 密码输入冲突

如果您遇到"输入密码会错乱"或类似错误，这意味着两个程序在同时尝试从终端读取输入。这正是我们推荐使用`detkey_ssh`函数而不是管道命令的原因。

## 技术实现

DetKey使用以下密码学组件：

1. **Argon2id**：将用户密码转换为高强度主种子
2. **HKDF**：从主种子衍生特定上下文的密钥种子
3. **密钥生成**：
   - **Ed25519**：从种子确定性密钥生成
   - **RSA**：使用HKDF输出作为确定性生成的熵源

### 密钥生成流程

```
主密码 → [Argon2id] → 主种子 → [HKDF + 上下文] → 熵 → [算法] → 密钥对
```

相同的主密码和上下文将始终在不同机器和不同时间段产生相同的密钥。

## 许可证

本项目遵循与仓库相同的许可证。

## 贡献

欢迎提交问题和拉取请求。在做出重大更改之前，请先开一个问题讨论您想要的更改。 