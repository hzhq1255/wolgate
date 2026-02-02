#!/usr/bin/env bash
#
# Wolgate 安装脚本 - 从 GitHub 最新 release 下载并安装
# 用法: curl -fsSL https://raw.githubusercontent.com/hzhq1255/wolgate/main/install/install.sh | bash
#   或: bash install.sh [-d DEST] [-v]
#

set -e

REPO="hzhq1255/wolgate"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
DEST="/usr/local/bin"
VERSION=""
DO_VERIFY=1
VERBOSE=0

# 解析参数
while getopts "d:nvh" opt; do
  case "$opt" in
    d) DEST="$OPTARG" ;;
    n) DO_VERIFY=0 ;;
    v) VERBOSE=1 ;;
    h)
      echo "用法: $0 [-d DEST] [-n] [-v]" >&2
      echo "  -d DEST  安装目录 (默认: /usr/local/bin)" >&2
      echo "  -n       跳过 sha256 校验" >&2
      echo "  -v       详细输出" >&2
      exit 0
      ;;
    *) echo "用法: $0 [-d DEST] [-n] [-v]" >&2; exit 1 ;;
  esac
done

log() {
  if [ "$VERBOSE" -eq 1 ]; then
    echo "[*] $*"
  fi
}

# 检测系统与架构，匹配 release 中的资产名
detect_asset_suffix() {
  local os kernel arch
  kernel=$(uname -s 2>/dev/null || true)
  arch=$(uname -m 2>/dev/null || true)

  case "$kernel" in
    Linux)  os="linux" ;;
    *)      echo "不支持的系统: $kernel (当前仅支持 Linux)" >&2; exit 1 ;;
  esac

  case "$arch" in
    x86_64|amd64)     echo "${os}-amd64" ;;
    i686|i386|x86)    echo "${os}-386" ;;
    aarch64|arm64)    echo "${os}-arm64" ;;
    armv6l)           echo "${os}-arm-v6" ;;
    armv7l|armhf)     echo "${os}-arm-v7" ;;
    mips)             echo "${os}-mips" ;;
    mipsle)           echo "${os}-mipsle" ;;
    mips64)           echo "${os}-mips64" ;;
    mips64le)         echo "${os}-mips64le" ;;
    *)                echo "不支持的架构: $arch" >&2; exit 1 ;;
  esac
}

# 需要 curl 或 wget
if command -v curl &>/dev/null; then
  GET="curl -fsSL"
elif command -v wget &>/dev/null; then
  GET="wget -qO-"
else
  echo "需要 curl 或 wget" >&2
  exit 1
fi

if command -v curl &>/dev/null; then
  DOWNLOAD="curl -fsSL -o"
else
  DOWNLOAD="wget -q -O"
fi

SUFFIX=$(detect_asset_suffix)
log "目标资产后缀: $SUFFIX"

# 获取最新 release 信息
echo "正在获取最新 release 信息..."
RELEASE_JSON=$($GET "$API_URL")
VERSION=$(echo "$RELEASE_JSON" | grep -oP '"tag_name":\s*"\K[^"]+' | head -1)
if [ -z "$VERSION" ]; then
  echo "无法获取最新版本" >&2
  exit 1
fi
echo "最新版本: $VERSION"

# 资产名: wolgate-1.0.0-linux-amd64 等
BASENAME="wolgate-${VERSION#v}-${SUFFIX}"
ASSET_URL=$(echo "$RELEASE_JSON" | grep -oP "\"browser_download_url\":\s*\"[^\"]*${BASENAME}\"" | head -1 | cut -d'"' -f4)
SHA_URL=$(echo "$RELEASE_JSON" | grep -oP "\"browser_download_url\":\s*\"[^\"]*${BASENAME}\.sha256\"" | head -1 | cut -d'"' -f4)

if [ -z "$ASSET_URL" ]; then
  echo "未找到匹配的二进制: $BASENAME" >&2
  exit 1
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
BINARY_PATH="$TMPDIR/$BASENAME"
FINAL_PATH="${DEST%/}/wolgate"

# 下载二进制
echo "正在下载 $BASENAME ..."
$DOWNLOAD "$BINARY_PATH" "$ASSET_URL"
chmod +x "$BINARY_PATH"

# 可选：校验 sha256
if [ "$DO_VERIFY" -eq 1 ] && [ -n "$SHA_URL" ]; then
  SHA_PATH="$TMPDIR/$BASENAME.sha256"
  $DOWNLOAD "$SHA_PATH" "$SHA_URL"
  (cd "$TMPDIR" && sha256sum -c "$BASENAME.sha256")
  log "sha256 校验通过"
elif [ "$DO_VERIFY" -eq 1 ]; then
  echo "未找到 sha256 文件，跳过校验"
fi

# 安装到目标目录
mkdir -p "$DEST"
if [ -w "$DEST" ]; then
  mv "$BINARY_PATH" "$FINAL_PATH"
else
  echo "需要 sudo 写入 $DEST"
  sudo mv "$BINARY_PATH" "$FINAL_PATH"
fi

echo "已安装: $FINAL_PATH"
"$FINAL_PATH" -version 2>/dev/null || "$FINAL_PATH" --version 2>/dev/null || true
