#!/bin/sh
#
# DetKey - 一键安装脚本
#
# 该脚本会自动检测您的操作系统和CPU架构，
# 并从 GitHub Releases 下载最新的、对应的 detkey 版本进行安装。
#
# 使用方法:
# curl -sfL https://raw.githubusercontent.com/lisonyang/the-axiom/main/install.sh | sh
#

set -e # 如果任何命令失败，则立即退出

# --- 配置 ---
GITHUB_USER="lisonyang"
GITHUB_REPO="the-axiom"

BINARY_NAME="detkey"
INSTALL_DIR="/usr/local/bin"

# --- 脚本开始 ---

echo "欢迎使用 DetKey 安装脚本！"
echo "DetKey - 确定性SSH密钥生成器"

# 1. 检测操作系统和架构
OS_TYPE=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH_TYPE=$(uname -m)

case $ARCH_TYPE in
  x86_64|amd64)
    ARCH_TYPE="amd64"
    ;;
  aarch64|arm64)
    ARCH_TYPE="arm64"
    ;;
  *)
    echo "错误：不支持的 CPU 架构: $ARCH_TYPE"
    echo "支持的架构: amd64, arm64"
    exit 1
    ;;
esac

echo "检测到您的系统: ${OS_TYPE}-${ARCH_TYPE}"

# 2. 获取最新的 Release 版本号
echo "正在从 GitHub 获取最新版本号..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/${GITHUB_USER}/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
  echo "错误：无法获取最新的 Release 版本号。"
  echo "请检查："
  echo "1. 仓库是否存在并且是公开的"
  echo "2. 是否已经创建了 Release"
  echo "3. 网络连接是否正常"
  exit 1
fi

echo "最新版本为: $LATEST_TAG"

# 3. 构建下载链接和文件名
ARCHIVE_EXT=".tar.gz"
if [ "$OS_TYPE" = "windows" ]; then
  ARCHIVE_EXT=".zip"
fi

DOWNLOAD_URL="https://github.com/${GITHUB_USER}/${GITHUB_REPO}/releases/download/${LATEST_TAG}/detkey-${LATEST_TAG}-${OS_TYPE}-${ARCH_TYPE}${ARCHIVE_EXT}"

# 4. 下载并解压
echo "正在下载: $DOWNLOAD_URL"

# 创建一个临时目录进行操作
TMP_DIR=$(mktemp -d)

# 下载文件
if ! curl -L --progress-bar "$DOWNLOAD_URL" -o "${TMP_DIR}/detkey-archive"; then
  echo "错误：下载失败。请检查："
  echo "1. 网络连接是否正常"
  echo "2. 该版本是否包含您的平台 (${OS_TYPE}-${ARCH_TYPE})"
  echo "可用的平台通常包括: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64"
  rm -rf "$TMP_DIR"
  exit 1
fi

echo "正在解压文件..."
if [ "$ARCHIVE_EXT" = ".zip" ]; then
  if command -v unzip >/dev/null 2>&1; then
    unzip -q "${TMP_DIR}/detkey-archive" -d "${TMP_DIR}"
  else
    echo "错误：需要 unzip 命令来解压 ZIP 文件。"
    rm -rf "$TMP_DIR"
    exit 1
  fi
else
  tar -xzf "${TMP_DIR}/detkey-archive" -C "${TMP_DIR}"
fi

# 5. 检查解压后的文件
BINARY_PATH="${TMP_DIR}/${BINARY_NAME}"
if [ "$OS_TYPE" = "windows" ]; then
  BINARY_PATH="${TMP_DIR}/${BINARY_NAME}.exe"
fi

if [ ! -f "$BINARY_PATH" ]; then
  echo "错误：解压后未找到 ${BINARY_NAME} 可执行文件。"
  ls -la "$TMP_DIR"
  rm -rf "$TMP_DIR"
  exit 1
fi

# 6. 安装到系统路径
echo "正在安装 detkey 到 ${INSTALL_DIR}..."

# 检查安装目录是否存在，以及是否需要 sudo
if [ ! -d "$INSTALL_DIR" ]; then
  echo "安装目录 ${INSTALL_DIR} 不存在。"
  if [ "$(id -u)" != "0" ]; then
    echo "尝试使用 sudo 创建..."
    sudo mkdir -p "$INSTALL_DIR"
  else
    mkdir -p "$INSTALL_DIR"
  fi
fi

# 安装二进制文件
if [ -w "$INSTALL_DIR" ]; then
  # 如果目录可写，直接移动
  mv "$BINARY_PATH" "${INSTALL_DIR}/${BINARY_NAME}"
  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
else
  # 如果目录不可写，尝试使用 sudo
  echo "需要管理员权限来安装到 ${INSTALL_DIR}..."
  sudo mv "$BINARY_PATH" "${INSTALL_DIR}/${BINARY_NAME}"
  sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
fi

# 7. 清理并验证
rm -rf "$TMP_DIR"

if command -v $BINARY_NAME >/dev/null 2>&1; then
  echo "✅ DetKey 安装成功！"
  echo ""
  echo "现在您可以在任何地方运行 'detkey' 命令了。"
  echo ""
  echo "使用示例:"
  echo "  # 生成私钥"
  echo "  detkey --context \"ssh/server-a/v1\""
  echo ""
  echo "  # 生成公钥"
  echo "  detkey --context \"ssh/server-a/v1\" --pub"
  echo ""
  echo "  # 查看帮助"
  echo "  detkey --help"
  echo ""
  echo "更多信息请访问: https://github.com/${GITHUB_USER}/${GITHUB_REPO}"
else
  echo "❌ 安装失败。"
  echo "请检查："
  echo "1. ${INSTALL_DIR} 是否在您的 PATH 中"
  echo "2. 是否有足够的权限"
  echo "3. 您可以尝试手动将二进制文件移动到 PATH 中的目录"
  exit 1
fi 