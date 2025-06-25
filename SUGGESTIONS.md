# DetKey 项目优化建议

## 1. 概述

`detkey` 是一个设计精良、文档齐全的优秀工具，其核心思想（确定性密钥生成）具有很高的实用价值。本项目在密码学应用、命令行设计和发布流程上已经做得非常出色。

本文档旨在基于现有代码和结构，提出一系列优化建议，目标是进一步提升项目的**用户体验、代码质量、可维护性和自动化程度**。

---

## 2. 功能增强建议

### 2.1. 集成mTLS证书签名功能

**现状:**
目前 `detkey` 能够生成用于 mTLS 的私钥，但用户仍需手动执行多个 `openssl` 命令来创建CA、签发服务器证书和客户端证书。这个过程繁琐且容易出错。

**建议:**
扩展 `detkey` 的功能，使其能够直接处理证书签名的整个流程。可以引入新的子命令或标志来实现。

**示例:**
```bash
# 1. 生成CA私钥并立即创建自签名CA证书
detkey --context "mtls/ca/v1" --type rsa4096 --action create-ca-cert --subj "/CN=My CA" > ca.crt

# 2. 使用CA签发服务器证书
# detkey 会在内部重新生成CA私钥用于签名
detkey --context "mtls/server/api.example.com/v1" \
       --type rsa4096 \
       --action sign-cert \
       --ca-context "mtls/ca/v1" \
       --subj "/CN=api.example.com" > server.crt
```

**优势:**
- **极大简化用户操作:** 将多步 `openssl` 操作简化为单行命令。
- **提升安全性:** 无需将CA私钥显式地保存在磁盘上，签名过程在内存中完成。
- **增强工具价值:** 使 `detkey` 成为一个完整的 mTLS 证书管理工具。

### 2.2. 提供Shell集成脚本

**现状:**
`README` 中提供的 `detkey_ssh` shell 函数非常实用，完美解决了 `ssh` 和 `detkey` 同时读取终端的冲突问题。但需要用户手动复制粘贴到他们的 shell 配置文件中。

**建议:**
增强 `install.sh` 脚本，在安装结束后，询问用户是否愿意将 `detkey_ssh` 函数自动添加到他们的 shell 配置文件（如 `.bashrc`, `.zshrc`）中。

**示例 `install.sh` 逻辑:**
```shell
# ... 安装二进制文件后 ...
read -p "Do you want to add the 'detkey_ssh' helper function to your shell config? (y/N) " choice
case "$choice" in 
  y|Y ) 
    echo "Adding function to ~/.zshrc and/or ~/.bashrc..."
    # 此处添加追加函数到配置文件的逻辑
    echo "Please run 'source ~/.zshrc' or 'source ~/.bashrc' to apply changes."
    ;;
  * ) 
    echo "Skipping. You can add the function manually later."
    ;;
esac
```

**优势:**
- **降低使用门槛:** 用户无需理解如何修改 shell 配置即可使用此高级功能。
- **提升用户体验:** 提供无缝的“开箱即用”体验。

---

## 3. 代码质量与可维护性

### 3.1. 项目结构重构

**现状:**
所有核心逻辑都集中在 `main.go` 一个文件中。对于当前规模尚可接受，但随着功能增加（如集成证书签名），会变得难以维护和测试。

**建议:**
采用标准的 Go 项目布局，将代码拆分到不同的包中。

**建议的目录结构:**
```
/
├── cmd/
│   └── detkey/
│       └── main.go      # CLI入口，负责解析参数和调用业务逻辑
├── internal/
│   ├── crypto/          # 核心加密逻辑 (deriveAndGenerateKey, deterministicReader)
│   ├── output/          # 处理不同格式的输出 (SSH, PEM)
│   └── mtls/            # mTLS证书生成和签名逻辑
├── pkg/
│   └── sshutil/         # 存放 detkey_ssh 函数的定义和说明，方便用户查看
├── go.mod
├── Makefile
└── ...
```

**优势:**
- **高内聚，低耦合:** 每个包都有明确的职责。
- **可测试性:** 可以对 `crypto`, `output` 等内部包编写独立的单元测试。
- **可维护性:** 新功能可以更容易地添加到相应的模块中。

### 3.2. 增加单元测试和集成测试

**现状:**
`Makefile` 中有 `test` 目标，但项目中缺少 `*_test.go` 文件。这意味着核心的密钥生成逻辑没有自动化测试覆盖。

**建议:**
- **单元测试:** 为 `internal/crypto` 包编写单元测试，确保对于固定的 `(password, salt, context)` 输入，总能生成确定性的、格式正确的密钥。
- **集成测试:** 编写一个简单的测试脚本或Go测试，调用编译好的 `detkey` 二进制文件，模拟真实的用户输入（通过命令行参数和stdin），并验证其输出是否符合预期。

**优势:**
- **保证代码质量:** 确保重构或添加新功能时不会破坏现有逻辑。
- **自动化验证:** 可以在CI流程中自动运行测试，防止有问题的代码被合并。

### 3.3. 使 `SALT` 可配置

**现状:**
`SALT` 在代码中被硬编码为一个常量。注释中提到 "Ideally, each user should use their own unique salt."

**建议:**
允许用户覆盖默认的 `SALT`。推荐的优先级顺序：
1. 命令行参数: `--salt "my-custom-salt"`
2. 环境变量: `DETKEY_SALT="my-custom-salt"`
3. 配置文件: `~/.config/detkey/config.toml`
4. 硬编码的默认值 (如果以上均未提供)

**优势:**
- **提升安全性:** 允许用户使用私有 salt，进一步增强密钥的安全性，即使 `detkey` 的默认 salt 泄露也不会影响他们。
- **灵活性:** 为高级用户提供更多自定义选项。

---

## 4. 构建与发布流程

### 4.1. 引入 GoReleaser

**现状:**
`Makefile` 和 `.github/workflows/release.yml` 共同完成跨平台构建和发布，功能完善但略显复杂。

**建议:**
使用 [GoReleaser](https://goreleaser.com/) 来统一和简化整个发布流程。GoReleaser 是 Go 社区构建和发布的黄金标准。

**使用 GoReleaser 的配置 (`.goreleaser.yml`):**
```yaml
builds:
  - main: ./cmd/detkey/main.go
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}
archives:
  - format: tar.gz
    # ... 其他配置
checksum:
  name_template: 'checksums.txt'
snapshot:
  name_template: "{{ incpatch .Version }}-next"
changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
```

**优势:**
- **简化配置:** 单个 `.goreleaser.yml` 文件可以替代大部分 `Makefile` 和 `release.yml` 的逻辑。
- **功能强大:** 自动处理构建、打包、生成校验和、创建GitHub Release、生成更新日志等。
- **社区标准:** 遵循 Go 社区的最佳实践。

### 4.2. 自动化版本发布说明

**现状:**
GitHub Release 的说明是静态的。

**建议:**
利用 GoReleaser 或其他工具（如 `git-chglog`）根据 Git 的提交历史自动生成Changelog，并填充到 Release 说明中。这需要团队遵循一定的 commit message 规范（如 Conventional Commits）。

**优势:**
- **自动化:** 减少手动编写发布说明的工作量。
- **信息准确:** 发布说明直接反映了两次发布之间的所有代码变更。

---

## 5. 文档优化

### 5.1. 增加“快速上手”章节

**现状:**
`README` 内容非常详尽，但对于想快速尝试的新用户来说可能有点长。

**建议:**
在 `README` 的最顶部增加一个“快速上手”（Quick Start）章节，用 3-4 个最核心的命令让用户在 1 分钟内体验到工具的价值。

**示例:**
```markdown
## Quick Start

1. **Install:**
   ```sh
   curl -sfL https://raw.githubusercontent.com/lisonyang/detkey/main/install.sh | sh
   ```

2. **Generate your first SSH private key:**
   ```sh
   # Enter a master password when prompted
   detkey --context "ssh/my-first-server/v1"
   ```

3. **Generate the corresponding public key:**
   ```sh
   detkey --context "ssh/my-first-server/v1" --pub
   ```
```

**优势:**
- **吸引用户:** 快速展示核心价值，降低初次使用的认知负担。
