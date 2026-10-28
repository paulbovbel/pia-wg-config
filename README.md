# pia-wg-config

A fast, portable Wireguard config generator for Private Internet Access (PIA) VPN.

[![Go Version](https://img.shields.io/github/go-mod/go-version/kylegrantlucas/pia-wg-config)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🌍 Region Selection (NOT Hardcoded!)

**IMPORTANT:** This tool supports ALL PIA regions through the `-r/--region` flag. The default region is `us_california`, but you can easily connect to any region:

```bash
# Connect to different regions
pia-wg-config -r uk_london USERNAME PASSWORD
pia-wg-config -r de_frankfurt USERNAME PASSWORD  
pia-wg-config -r au_sydney USERNAME PASSWORD
pia-wg-config -r japan USERNAME PASSWORD
```

### List All Available Regions

To see all available regions before connecting:

```bash
pia-wg-config regions
```

This will show you the complete list of PIA server regions you can connect to.

## 🚀 Quick Start

### Install Go

#### Linux

```bash
sudo apt-get update && sudo apt-get install golang-go

# define go installation location and add to path
go env -w GOPATH=$HOME/go
echo "export GOPATH=\$HOME/go" >> ~/.bash_profile
echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.bash_profile
source ~/.bash_profile
```

#### macOS

```bash
brew install go

# define go installation location and add to path
go env -w GOPATH=$HOME/go
echo "export GOPATH=\$HOME/go" >> ~/.zshenv
echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.zshenv
source ~/.zshenv
```

### Install pia-wg-config

```bash
go install github.com/kylegrantlucas/pia-wg-config@latest
```

### Basic Usage

```bash
# Generate config for default region (us_california)
pia-wg-config USERNAME PASSWORD

# Generate config for a specific region
pia-wg-config -r uk_london USERNAME PASSWORD

# Save config to file
pia-wg-config -o wg0.conf -r de_frankfurt USERNAME PASSWORD

# Enable verbose output
pia-wg-config -v -r japan USERNAME PASSWORD
```

## 📖 Command Reference

### Main Command

```
pia-wg-config [OPTIONS] USERNAME PASSWORD
```

**Arguments:**
- `USERNAME` - Your PIA username
- `PASSWORD` - Your PIA password

**Options:**
- `-r, --region` - Region to connect to (default: "us_california")
- `-o, --outfile` - Output file for the config (default: stdout)
- `-p, --port-forwarding` - Fail unless the region offers port forwarding
- `--server-name-file` - Write the selected server's TLS name (see Port forwarding)
- `--gateway-file` - Write the selected server's in-tunnel address (see Port forwarding)
- `-v, --verbose` - Enable verbose output
- `-h, --help` - Show help

### Port forwarding

PIA lists several WireGuard servers per region, and a given server may refuse a
key or be missing the port forwarding service entirely. Registration tries the
region's servers in a random order until one accepts, so a broken server no
longer blocks every attempt. Pass `-p` to refuse regions that do not offer
forwarding at all.

Two files help a downstream client such as [Gluetun](https://github.com/qdm12/gluetun)
reach PIA's forwarding API, which listens on port 19999 inside the tunnel:

- `--server-name-file` holds the TLS name of the server that accepted the key,
  which the API's certificate is issued for.
- `--gateway-file` holds that server's in-tunnel address (`server_vip`). It is
  not derivable from the peer address, so a client that does not know it has to
  guess, and a wrong guess looks like an unreachable API.

```bash
pia-wg-config -r ca_toronto -p \
  -o wg0.conf \
  --server-name-file server-name \
  --gateway-file gateway
```

### Subcommands

- `pia-wg-config regions` - List all available PIA regions

## 🌐 Popular Regions

Here are some commonly used region codes:

| Region Code | Location |
|-------------|----------|
| `us_california` | United States - California |
| `us_east` | United States - East Coast |
| `uk_london` | United Kingdom - London |
| `de_frankfurt` | Germany - Frankfurt |
| `ca_toronto` | Canada - Toronto |
| `au_sydney` | Australia - Sydney |
| `japan` | Japan |
| `singapore` | Singapore |
| `netherlands` | Netherlands |
| `sweden` | Sweden |

Run `pia-wg-config regions` for the complete list.

## 💡 Usage Examples

### Connect to UK servers
```bash
pia-wg-config -r uk_london -o uk-wg.conf myusername mypassword
sudo wg-quick up uk-wg.conf
```

### Connect to German servers
```bash
pia-wg-config -r de_frankfurt -o germany.conf myusername mypassword
sudo wg-quick up germany.conf
```

### Quick connection (output to stdout)
```bash
pia-wg-config -r netherlands myusername mypassword > vpn.conf
```

## 🔧 Integration Examples

### Bash Script for Multiple Regions
```bash
#!/bin/bash
REGIONS=("uk_london" "de_frankfurt" "us_california" "japan")
USERNAME="your_username"
PASSWORD="your_password"

for region in "${REGIONS[@]}"; do
    echo "Generating config for $region..."
    pia-wg-config -r "$region" -o "configs/${region}.conf" "$USERNAME" "$PASSWORD"
done
```

### Docker Usage
```dockerfile
FROM golang:alpine AS builder
RUN go install github.com/kylegrantlucas/pia-wg-config@latest

FROM alpine:latest
RUN apk --no-cache add ca-certificates wireguard-tools
COPY --from=builder /go/bin/pia-wg-config /usr/local/bin/
ENTRYPOINT ["pia-wg-config"]
```

## 🏗️ Building from Source

```bash
git clone https://github.com/kylegrantlucas/pia-wg-config
cd pia-wg-config
go build -o pia-wg-config .
```

## 🐛 Troubleshooting

### Common Issues

**"Region not found" error:**
- Run `pia-wg-config regions` to see available regions
- Check your spelling of the region code
- Region codes are case-sensitive

**Authentication errors:**
- Verify your PIA username and password
- Make sure your PIA subscription is active
- Try connecting through the PIA app first to verify credentials

**Network connectivity issues:**
- Check your internet connection
- Try with verbose mode: `-v` flag
- Some networks block VPN traffic

### Getting Help

1. Check the [Issues](https://github.com/kylegrantlucas/pia-wg-config/issues) page
2. Run with `-v` flag for detailed output
3. Verify your PIA account is active

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup
```bash
git clone https://github.com/kylegrantlucas/pia-wg-config
cd pia-wg-config
go mod download
go test ./...
```

## 📋 Requirements

- Go 1.27 or later (for building; older Go fails with a `go.mod requires go >= 1.27` error — upgrade Go or set `GOTOOLCHAIN=auto`)
- Active PIA subscription
- Wireguard client (for using generated configs)

## 🔐 Security

For automation, supply `PIA_USERNAME` and `PIA_PASSWORD` in the process environment
(e.g. from a secret-manager-provided systemd `EnvironmentFile`). They take precedence
over positional credentials; both must be nonempty. Do not expand them into command
arguments or place plaintext credentials in unit definitions. Environment variables
remain readable by privileged users.

```bash
# PIA_USERNAME and PIA_PASSWORD are already supplied by the service manager.
pia-wg-config -r ca_toronto -o /run/pia-wireguard/wg0.conf \
  --server-name-file /run/pia-wireguard/server-name
```

`--server-name-file` writes the selected server's TLS name for Gluetun's
`SERVER_NAMES` setting. Use a private runtime directory and generate both outputs
before starting Gluetun. The PIA CA download is pinned to an upstream commit, and
the config uses PIA's returned server port. Verbose mode never logs private keys.
Without `-o`, stdout contains the private config: do not send it to logs.

- This tool connects directly to PIA's official API endpoints
- Your credentials are only used for authentication and are not stored
- Generated configs contain your unique keys - keep them secure
- Configs expire and need to be regenerated periodically

## 📚 Background

Based on the [manual-connections](https://github.com/pia-foss/manual-connections) scripts provided by Private Internet Access. This Go implementation provides:

- **Portability** - Single binary that runs anywhere
- **Speed** - Fast config generation
- **Reliability** - No external dependencies
- **Simplicity** - Easy command-line interface

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⭐ Star History

If this tool helped you, consider giving it a star! It helps others discover the project.

---

**Note for Forkers:** You don't need to fork this repository to change regions! Use the `-r` flag instead. This tool supports all PIA regions out of the box.
