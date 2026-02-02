# Wolgate 安装脚本

从 [GitHub Releases](https://github.com/hzhq1255/wolgate/releases) 最新版本自动下载并安装 wolgate。

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/hzhq1255/wolgate/main/install/install.sh | bash
```

或指定安装目录（当前目录）：

```bash
curl -fsSL https://raw.githubusercontent.com/hzhq1255/wolgate/main/install/install.sh | bash -s -- -d .
```

## 本地运行

```bash
chmod +x install.sh
./install.sh              # 安装到 /usr/local/bin
./install.sh -d .         # 安装到当前目录
./install.sh -n           # 跳过 sha256 校验
./install.sh -v            # 详细输出
```

## 选项

| 选项 | 说明 |
|------|------|
| `-d DEST` | 安装目录，默认 `/usr/local/bin` |
| `-n` | 不校验 sha256 |
| `-v` | 显示详细日志 |
| `-h` | 显示帮助 |

## 要求

- Linux（与 [release](https://github.com/hzhq1255/wolgate/releases) 中提供的二进制一致）
- `curl` 或 `wget`
- 写入目标目录的权限（如需安装到 `/usr/local/bin` 会使用 sudo）

## 支持的平台

脚本会根据 `uname -m` 自动选择对应二进制：

- x86_64 → linux-amd64
- i686/i386 → linux-386
- aarch64 → linux-arm64
- armv6l → linux-arm-v6
- armv7l → linux-arm-v7
- mips / mipsle / mips64 / mips64le
