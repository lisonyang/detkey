# 发布指南

本文档说明如何为 DetKey 项目发布新版本。

## 自动化发布流程

项目已配置了 GitHub Actions 自动化发布流程，当推送带有版本标签的提交时，会自动构建并发布多平台二进制文件。

## 发布步骤

### 1. 准备发布

确保您的代码已经测试并准备好发布：

```bash
# 运行测试
go test ./...

# 确保代码能正常构建
go build -o detkey .

# 测试基本功能
./detkey --context "test/v1" --pub
```

### 2. 创建并推送版本标签

使用语义化版本号创建标签：

```bash
# 创建版本标签（例如 v1.0.0）
git tag v1.0.0

# 推送标签到 GitHub
git push origin v1.0.0
```

### 3. 自动化构建和发布

推送标签后，GitHub Actions 会自动：

1. **构建多平台二进制文件**：
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64)

2. **打包文件**：
   - Linux/macOS: `.tar.gz` 格式
   - Windows: `.zip` 格式

3. **创建 GitHub Release**：
   - 自动生成发布说明
   - 上传所有平台的二进制包
   - 包含安装指令

### 4. 验证发布

发布完成后，您可以：

1. 检查 [Releases 页面](https://github.com/lisonyang/the-axiom/releases)
2. 测试一键安装脚本：
   ```bash
   curl -sfL https://raw.githubusercontent.com/lisonyang/the-axiom/main/install.sh | sh
   ```

## 版本号规范

建议使用语义化版本号：

- `v1.0.0` - 主要版本（破坏性更改）
- `v1.1.0` - 次要版本（新功能）
- `v1.1.1` - 修补版本（错误修复）

## 支持的平台

当前自动化构建支持以下平台：

- `linux-amd64` - Linux 64位
- `linux-arm64` - Linux ARM64
- `darwin-amd64` - macOS Intel
- `darwin-arm64` - macOS Apple Silicon
- `windows-amd64` - Windows 64位

## 故障排除

### 构建失败

如果 GitHub Actions 构建失败：

1. 检查 Actions 页面的日志
2. 确保代码能在本地正常构建
3. 检查 Go 版本兼容性

### 发布不包含某个平台

如果某个平台的二进制文件缺失：

1. 检查 `.github/workflows/release.yml` 中的构建矩阵
2. 确保没有排除该平台
3. 检查构建日志中的错误

### 安装脚本失败

如果用户反馈安装脚本失败：

1. 检查 GitHub API 是否正常
2. 确认发布中包含对应平台的文件
3. 检查文件命名是否符合脚本期望

## 手动发布

如果需要手动发布（不推荐），可以：

1. 本地构建所有平台：
   ```bash
   make build-all  # 如果有 Makefile
   ```

2. 手动创建 Release 并上传文件

## 联系信息

如有问题，请在 GitHub 上创建 Issue。 