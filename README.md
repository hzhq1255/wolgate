# Wolgate

Lightweight Wake-on-LAN gateway with web management interface.

## Features

- **Web Management UI** - Intuitive interface for managing devices
- **ARP Discovery** - Import devices from local ARP table
- **WOL Magic Packet** - Send Wake-on-LAN packets to network devices
- **Device Management** - Add, edit, delete, and organize devices into groups
- **RESTful API** - JSON API for programmatic access
- **Data Persistence** - Device data stored in JSON file
- **Logging** - Configurable log levels and file rotation

## Installation

### One-Line Install (Recommended)

Automatically downloads and installs the latest release for your platform:

```bash
curl -fsSL https://raw.githubusercontent.com/hzhq1255/wolgate/main/install/install.sh | bash
```

Or with wget:

```bash
wget -qO- https://raw.githubusercontent.com/hzhq1255/wolgate/main/install/install.sh | bash
```

**Install script options:**

```bash
bash install/install.sh -d /usr/local/bin  # Install to custom directory
bash install/install.sh -n                 # Skip SHA256 verification
bash install/install.sh -v                 # Verbose output
```

**Supported platforms:**
- Linux: x86_64, i386
- Linux ARM: arm64, arm v6, arm v7
- Linux MIPS: mips, mipsle, mips64, mips64le

### Manual Binary Download

Download the latest release from [Releases](https://github.com/hzhq1255/wolgate/releases).

1. Download the binary for your platform
2. Verify the checksum: `sha256sum -c wolgate-*.sha256`
3. Make executable: `chmod +x wolgate-*`
4. Move to PATH: `sudo mv wolgate-* /usr/local/bin/wolgate`

### From Source

```bash
git clone https://github.com/hzhq1255/wolgate.git
cd wolgate
go build -o wolgate .
```

## Quick Start

### Start Web Server

```bash
./wolgate -config ./wolgate.json server -listen 0.0.0.0:9000 -data ./wolgate_data.json
```

Then open http://localhost:9000 in your browser.

### Send WOL Packet via CLI

```bash
./wolgate wake -mac AA:BB:CC:DD:EE:FF
```

## Configuration

Create `wolgate.json`:

```json
{
  "server": {
    "listen": "0.0.0.0:9000",
    "data": "./wolgate_data.json"
  },
  "wake": {
    "iface": "",
    "broadcast": "255.255.255.255"
  },
  "log": {
    "file": "",
    "level": "info",
    "max_size": 10,
    "max_backups": 3,
    "max_age": 7
  }
}
```

## Commands

### server

Start the web management service.

```bash
./wolgate server [options]

Options:
  -listen string   HTTP listen address (default from config)
  -data string     Device data file path (default from config)
  -iface string    Network interface for WOL (default from config)
```

### wake

Send a WOL magic packet to a device.

```bash
./wolgate wake -mac <MAC> [options]

Options:
  -mac string     Target MAC address (required)
  -iface string   Network interface
  -bcast string   Broadcast address
```

### version

Show version information.

```bash
./wolgate version
```

## API Endpoints

### Devices

- `GET /api/devices` - List all devices
- `POST /api/devices` - Add a new device
- `PUT /api/devices/:id` - Update a device
- `DELETE /api/devices/:id` - Delete a device

### WOL

- `POST /api/wake/:id` - Send WOL packet to device

### ARP

- `GET /api/arp` - List ARP table entries

## Project Structure

```
wolgate/
├── arp/        # ARP table parsing
├── config/     # Configuration management
├── logger/     # Logging utilities
├── store/      # Device data storage
├── web/        # Web UI and HTTP API
├── wol/        # Wake-on-LAN packet sender
└── main.go     # CLI entry point
```

## Development

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -ldflags "-X main.Version=1.0.0" -o wolgate .
```

## License

MIT License
