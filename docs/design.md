# Wolgate 设计与开发方案

## 1. 项目概述

**wolgate** 是一个面向小型路由器与内网环境的轻量级 Wake-on-LAN 管理工具，既支持 Web 服务方式集中管理内网设备，也支持命令行方式直接唤醒单台设备。

### 设计目标

- 单一二进制文件
- 低内存占用（10-20MB）
- 可运行于嵌入式/路由器环境
- 支持网页管理与 CLI 工具双模式
- 通过 Nginx 反向代理实现认证与安全控制

---

## 2. 运行环境约束

- **设备类型**: 小型路由器（ARM / MIPS）
- **可写存储**: `/data`、`/tmp`
- **内存**: ~400MB（需极低常驻占用）
- **RootFS**: 空间紧张，不依赖运行时环境

---

## 3. 总体架构

```
Browser
  |
  |  HTTPS + Basic Auth
  v
Nginx
  |
  |  /wol/
  v
wolgate (Go 单二进制)
  |
  +-- HTTP Server
  +-- HTML (embed)
  +-- JSON Storage
  +-- ARP Import
  +-- WOL UDP Sender
```

---

## 4. 子命令设计

### 4.1 CLI 总体结构

```bash
wolgate <command> [options]

Commands:
  server    启动 Web 管理服务
  wake      直接发送 WOL 魔术包
  version   显示版本信息

Global Options:
  -config     配置文件路径
  -log        日志文件路径
  -log-level  日志级别
```

### 4.2 server 子命令

用于常驻运行，通过 Nginx 反向代理访问。

**示例:**

```bash
wolgate server -listen 127.0.0.1:9000 -data /data/wolgate.json
```

**参数:**

| 参数 | 说明 |
|------|------|
| `-listen` | HTTP 监听地址 |
| `-data` | 设备数据存储路径 |
| `-iface` | 广播网卡（可选） |

### 4.3 wake 子命令

用于命令行或脚本直接唤醒设备。

**示例:**

```bash
wolgate wake -mac AA:BB:CC:DD:EE:FF
```

**参数:**

| 参数 | 说明 |
|------|------|
| `-mac` | 目标 MAC（必填） |
| `-iface` | 网络接口（可选） |
| `-bcast` | 广播地址（可选） |

---

## 5. 配置文件设计

JSON 格式，轻量易解析。

```json
{
  "server": {
    "listen": "127.0.0.1:9000",
    "data": "/data/wolgate.json"
  },
  "wake": {
    "iface": "br-lan",
    "broadcast": "192.168.31.255"
  },
  "log": {
    "file": "/tmp/wolgate.log",
    "level": "info",
    "max_size": 10,
    "max_backups": 3,
    "max_age": 7
  }
}
```

### 配置优先级

```
命令行参数 > 环境变量 > 配置文件 > 默认值
```

---

## 6. 模块划分

| 模块 | 功能 |
|------|------|
| `main` | 子命令解析、启动流程 |
| `config` | 配置加载与合并 |
| `store` | 设备数据 JSON 存储 |
| `wol` | WOL 魔术包发送 |
| `arp` | ARP 表解析与导入 |
| `web` | HTTP API 与 HTML 页面 |
| `logger` | 日志管理 |

---

## 7. API 路由设计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | / | HTML 页面 |
| GET | /api/devices | 设备列表 |
| POST | /api/devices | 添加设备 |
| PUT | /api/devices/:id | 更新设备 |
| DELETE | /api/devices/:id | 删除设备 |
| POST | /api/wake/:id | 唤醒设备 |
| GET | /api/arp | 获取 ARP 表设备 |
| POST | /api/import | 批量导入设备 |

---

## 8. 数据存储

- **存储格式**: JSON
- **默认路径**: `/data/wolgate.json`
- **策略**: 启动时加载，修改即写回

---

## 9. 安全模型

- wolgate 不内置认证
- 通过 Nginx 实现 Basic Auth
- 仅监听 `127.0.0.1`

---

## 10. 日志设计

- 使用 Go 标准库 `log`
- 支持日志文件路径配置
- 支持日志轮转

**示例格式:**

```
2026-01-31T12:00:00 INFO wolgate started
```

---

## 11. 编译与部署

### 编译参数

```bash
# ARM
CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags "-s -w"

# MIPS
CGO_ENABLED=0 GOOS=linux GOARCH=mips go build -ldflags "-s -w"
```

### 运行

```bash
/tmp/wolgate server
```

---

## 12. 开发阶段规划

1. ✅ CLI 与子命令骨架
2. ✅ wake 子命令实现
3. ✅ server HTTP API
4. ✅ HTML 管理页面
5. ✅ ARP 导入
6. ⏳ 路由器实测与优化

---

## 13. 目标指标

| 指标 | 目标 |
|------|------|
| Binary 大小 | ≤ 3MB |
| 常驻内存 | ≤ 15MB |
| 外部依赖 | 零依赖 |

---

## 14. 总结

wolgate 定位为：**路由器上的轻量级内网唤醒网关**

同时兼顾：
- 工具化（CLI）
- 服务化（Web）
- 长期可维护性
