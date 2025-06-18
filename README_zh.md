# DetKey - 确定性SSH密钥生成器

[English](README.md) | [中文](README_zh.md)

DetKey 是一个强大的命令行工具，它允许您使用一个主密码和上下文字符串来确定性地生成SSH密钥。这意味着相同的输入总是会产生相同的密钥对，让您能够在任何地方重新生成相同的SSH密钥，而无需存储或传输密钥文件。

## 核心特性

- **确定性生成**: 相同的主密码和上下文总是生成相同的密钥对
- **无依赖**: 编译为单一可执行文件，无需任何外部依赖
- **跨平台**: 支持 Linux, macOS, Windows
- **安全设计**: 使用 Argon2id 进行密钥延伸，HKDF 进行密钥衍生
- **标准格式**: 输出标准的 OpenSSH 格式密钥

## 安装

### 一键安装（推荐）

您可以使用以下命令自动下载并安装最新版本：

```bash
curl -sfL https://raw.githubusercontent.com/lisonyang/detkey/main/install.sh | sh
```

该脚本会自动：
- 检测您的操作系统和 CPU 架构
- 从 GitHub Releases 下载对应的二进制文件
- 安装到 `/usr/local/bin` 目录（可能需要 sudo 权限）

### 手动安装

1. 访问 [Releases 页面](https://github.com/lisonyang/detkey/releases)
2. 下载适合您系统的压缩包
3. 解压并将 `detkey` 可执行文件移动到 PATH 中的目录

### 从源码构建

确保您已安装 Go 1.21 或更高版本，然后运行：

```bash
go mod tidy
go build -o detkey
```

#### 跨平台编译

为不同平台编译：

```bash
# Linux (AMD64)
GOOS=linux GOARCH=amd64 go build -o detkey-linux

# Windows (AMD64)
GOOS=windows GOARCH=amd64 go build -o detkey.exe

# macOS (ARM64)
GOOS=darwin GOARCH=arm64 go build -o detkey-darwin-arm64
```

## 使用方法

### 基本用法

```bash
# 生成私钥
./detkey --context "ssh/server-a/v1"

# 生成公钥
./detkey --context "ssh/server-a/v1" --pub
```

### 实际使用场景

**⚠️ 重要提示：可靠的SSH密钥部署方法**

常见的一行管道命令可能会导致密码输入冲突，因为 `detkey` 和 `ssh` 两个程序会同时尝试从终端读取密码。为了确保可靠的部署，请使用下面的**三步文件法**：

#### 1. 可靠的公钥部署方法

**第1步：生成公钥到临时文件**
```bash
# 生成公钥并保存到临时文件
./detkey --context "ssh/prod-server/v1" --pub > /tmp/prod-server.pub
```

**第2步：使用 ssh-copy-id 部署（推荐）**
```bash
# 使用 OpenSSH 官方推荐的部署工具
ssh-copy-id -i /tmp/prod-server.pub user@server
```

**第3步：清理临时文件**
```bash
# 删除临时文件
rm /tmp/prod-server.pub
```

**备选部署方法（如果没有 ssh-copy-id）：**
```bash
# 第1步：生成公钥到临时文件
./detkey --context "ssh/prod-server/v1" --pub > /tmp/prod-server.pub

# 第2步：手动部署
cat /tmp/prod-server.pub | ssh user@server "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && chmod 700 ~/.ssh"

# 第3步：清理临时文件
rm /tmp/prod-server.pub
```

#### 2. 使用生成的私钥登录

```bash
# 使用进程替换直接登录，私钥不会保存到磁盘（部署完成后此方法完全可靠）
ssh -i <(./detkey --context "ssh/prod-server/v1") user@server
```

#### 3. 创建方便的别名

在您的 `~/.bashrc` 或 `~/.zshrc` 中添加：

```bash
alias ssh-prod='ssh -i <(detkey --context "ssh/prod-server/v1") user@prod-server'
alias ssh-dev='ssh -i <(detkey --context "ssh/dev-server/v1") user@dev-server'
```

然后您就可以简单地运行：

```bash
ssh-prod  # 连接到生产服务器
ssh-dev   # 连接到开发服务器
```

#### 为什么三步文件法更可靠？

1. **完全隔离**: 每个需要输入密码的步骤都是独立、无干扰的。`detkey` 和 `ssh-copy-id` 在不同的时间点与您的终端进行交互。
2. **职责单一**: 我们让每个工具只做它最擅长的事：
   - `detkey`: 只负责生成密钥内容
   - `ssh-copy-id`: 只负责部署公钥文件，这是它的本职工作，能处理各种边缘情况
3. **无管道冲突**: 我们彻底避免了导致问题的根源——在同一命令行上混合使用多个需要交互式输入的程序

## 上下文字符串设计

上下文字符串用于区分不同的用途。建议使用有层次结构的命名：

```
ssh/production/web-server-1/v1
ssh/staging/database/v1
ssh/personal/vps/v2
git/github/personal/v1
git/gitlab/work/v1
```

## 安全考虑

### 优势

- **密钥延伸**: 使用 Argon2id 算法使暴力破解成本极高
- **隔离性**: 不同上下文生成完全独立的密钥
- **不存储**: 密钥在内存中生成，使用后即刻销毁
- **确定性**: 无需担心密钥丢失或备份

### 权衡

- **主密码强度**: 工具的安全性依赖于您主密码的强度
- **离线攻击**: 如果攻击者获得工具和一个已知的密钥对，可能尝试暴力破解主密码

### 最佳实践

1. **使用强主密码**: 建议使用包含大小写字母、数字和特殊字符的长密码
2. **保护工具安全**: 不要在不信任的环境中使用
3. **上下文版本控制**: 如需更换密钥，更改上下文中的版本号
4. **定期轮换**: 定期更换重要服务的密钥

## 故障排除

### 常见SSH部署问题

如果您在使用三步文件法时仍然遇到问题，请检查以下几点：

1. **服务器SSH配置**: 确保 `/etc/ssh/sshd_config` 允许公钥认证：
   ```
   PubkeyAuthentication yes
   AuthorizedKeysFile .ssh/authorized_keys
   ```

2. **文件权限**: 验证服务器上的权限设置：
   ```bash
   chmod 700 ~/.ssh
   chmod 600 ~/.ssh/authorized_keys
   ```

3. **SELinux/AppArmor**: 在某些系统上，安全模块可能会限制SSH密钥操作。如果部署失败，请检查系统日志。

4. **网络问题**: 在尝试基于密钥的认证之前，确保SSH密码认证连接正常工作。

### 密码输入冲突

如果您遇到"输入密码会错乱"或类似错误，这意味着两个程序在同时尝试从终端读取输入。这正是我们推荐使用三步文件法而不是管道命令的原因。

## 技术实现

DetKey 使用以下密码学组件：

1. **Argon2id**: 用于将用户密码转换为高强度主种子
2. **HKDF**: 用于从主种子衍生特定上下文的密钥种子
3. **Ed25519**: 用于生成SSH密钥对

### 密钥生成流程

```
主密码 → [Argon2id] → 主种子 → [HKDF + 上下文] → 最终种子 → [Ed25519] → SSH密钥对
```

## 许可证

本项目遵循与仓库相同的许可证。

## 贡献

欢迎提交问题和拉取请求。在做出重大更改之前，请先开一个问题讨论您想要的更改。 