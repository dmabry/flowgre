# Flowgre

Slinging packets since 2022!

```
    ___ _
  / __\ | _____      ____ _ _ __ ___
 / _\ | |/ _ \ \ /\ / / _` | '__/ _ \
 / /   | | (_) \ V  V / (_| | | |  __/
 \/    |_|\___/ \_/\_/ \__, |_|  \___|
                       |___/
```

For sending fabricated NetFlow v9 and IPFIX (RFC 7011) traffic to a collector for testing. Supports both IPv4 and IPv6 flow records with auto-detection from CIDR ranges.

[![Go Tests](https://github.com/dmabry/flowgre/actions/workflows/go-test.yml/badge.svg)](https://github.com/dmabry/flowgre/actions/workflows/go-test.yml)
[![Security Scan](https://github.com/dmabry/flowgre/actions/workflows/security.yml/badge.svg)](https://github.com/dmabry/flowgre/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmabry/flowgre)](https://goreportcard.com/report/github.com/dmabry/flowgre)
[![Go Reference](https://pkg.go.dev/badge/github.com/dmabry/flowgre.svg)](https://pkg.go.dev/github.com/dmabry/flowgre)

## Table of Contents
- [CLI Flags Reference](#cli-flags-reference)
- [Exit Codes](#exit-codes)
- [Configuration Keys Reference](#configuration-keys-reference)
- [Single Mode](#single-mode)
- [Barrage Mode](#barrage-mode)
- [IPFIX Mode](#ipfix-mode)
- [Record Mode](#record-mode)
- [Replay Mode](#replay-mode)
- [Proxy Mode](#proxy-mode)
- [Web Dashboard](#web-dashboard)
- [License](#license)

## CLI Flags Reference

Flowgre uses Go's `flag` package. Flags are passed with a single dash (`-flag`) and are scoped to individual subcommands. Global subcommands (`version`, `help`) take no flags.

### Global Subcommands

| Subcommand | Description |
|---|---|
| `version` | Print the current version and license. Overridden at build time via `-ldflags -X main.version`. |
| `help` | Print the help header with subcommand summaries. |

### `single` — Send sequential flows

Source: [`cmd/single.go`](cmd/single.go)

| Flag | Type | Default | Description |
|---|---|---|---|
| `-server` | string | `127.0.0.1` | Servername or IP address of flow collector (IPv4 or IPv6) |
| `-port` | int | `9995` | Destination port used by the flow collector |
| `-src-port` | int | `0` | Source port used by the client. If `0`, a random port between 10000–15000 is chosen |
| `-count` | int | `1` | Count of flows to send in sequence |
| `-hexdump` | bool | `false` | If true, do a hexdump of each packet |
| `-src-range` | string | `10.0.0.0/8` | CIDR range for source IPs (IPv4 or IPv6) |
| `-dst-range` | string | `10.0.0.0/8` | CIDR range for destination IPs (IPv4 or IPv6) |

### `barrage` — Continuous flow barrage

Source: [`cmd/barrage.go`](cmd/barrage.go)

| Flag | Type | Default | Description |
|---|---|---|---|
| `-server` | string | `127.0.0.1` | Servername or IP address of the flow collector (IPv4 or IPv6) |
| `-port` | int | `9995` | Destination port used by the flow collector |
| `-src-range` | string | `10.0.0.0/8` | CIDR range for source IPs (IPv4 or IPv6) |
| `-dst-range` | string | `10.0.0.0/8` | CIDR range for destination IPs (IPv4 or IPv6) |
| `-workers` | int | `4` | Number of workers to create. Each worker uses unique source addresses |
| `-delay` | int | `100` | Milliseconds between packets sent |
| `-template-interval` | int | `30` | Seconds between template retransmissions (`0` to disable) |
| `-config` | string | *(empty)* | Path to a YAML config file. Supersedes all other flags when provided |
| `-web` | bool | `false` | Enable the web dashboard server |
| `-web-ip` | string | `127.0.0.1` | IP address the web server listens on (IPv4 or IPv6) |
| `-web-port` | int | `8080` | Port to bind the web server on |
| `-web-username` | string | *(empty)* | Web server username (falls back to `FLOWGRE_WEB_USERNAME` env var, then `admin`) |
| `-web-password` | string | *(empty)* | Web server password (falls back to `FLOWGRE_WEB_PASSWORD` env var) |
| `-protocol` | string | `netflow` | Protocol to use: `netflow` or `ipfix` |
| `-profile` | string | `generic` | NetFlow flow profile: `generic`, `minimal`, or `extended` |

### `ipfix` — Send IPFIX flows

Source: [`cmd/ipfix_single.go`](cmd/ipfix_single.go)

| Flag | Type | Default | Description |
|---|---|---|---|
| `-server` | string | `127.0.0.1` | Servername or IP address of flow collector (IPv4 or IPv6) |
| `-port` | int | `9995` | Destination port used by the flow collector |
| `-src-port` | int | `0` | Source port used by the client. If `0`, a random port between 10000–15000 is chosen |
| `-count` | int | `1` | Count of flows to send in sequence |
| `-hexdump` | bool | `false` | If true, do a hexdump of each packet |
| `-src-range` | string | `10.0.0.0/8` | CIDR range for source IPs (IPv4 or IPv6) |
| `-dst-range` | string | `10.0.0.0/8` | CIDR range for destination IPs (IPv4 or IPv6) |

### `record` — Capture flows to disk

Source: [`cmd/record.go`](cmd/record.go)

| Flag | Type | Default | Description |
|---|---|---|---|
| `-ip` | string | `127.0.0.1` | IP address to listen on (IPv4 or IPv6) |
| `-port` | int | `9995` | Listen UDP port |
| `-db` | string | `recorded_flows` | Directory to place recorded flows for later replay |
| `-verbose` | bool | `false` | Log every packet received (warning: high volume) |

### `replay` — Replay recorded flows

Source: [`cmd/replay.go`](cmd/replay.go)

| Flag | Type | Default | Description |
|---|---|---|---|
| `-server` | string | `127.0.0.1` | Target server to replay flows at (IPv4 or IPv6) |
| `-port` | int | `9995` | Target server UDP port |
| `-delay` | int | `100` | Milliseconds between packets sent |
| `-db` | string | `recorded_flows` | Directory to read recorded flows from |
| `-loop` | bool | `false` | Loop the replays indefinitely |
| `-workers` | int | `1` | Number of concurrent workers for replay |
| `-updatets` | bool | `false` | Update timestamps on replayed flows to the current time |
| `-verbose` | bool | `false` | Log every packet sent (warning: high volume) |

### `proxy` — Relay flows to multiple targets

Source: [`cmd/proxy.go`](cmd/proxy.go)

| Flag | Type | Default | Description |
|---|---|---|---|
| `-ip` | string | `127.0.0.1` | IP address the proxy listens on (IPv4 or IPv6) |
| `-port` | int | `9995` | Proxy listen UDP port |
| `-target` | string | *(required)* | Target in `IP:PORT` format. Repeat this flag for multiple targets |
| `-verbose` | bool | `false` | Log every flow received (warning: high volume) |

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| `0` | Success | Normal termination, graceful shutdown |
| `1` | Error | Invalid arguments, parse failure, network error, or runtime panic (`log.Fatal`/`log.Fatalf`) |
| `2` | Unknown subcommand | Passed unrecognized subcommand to `main` |

**Details:**

- **Exit 1** is used broadly across all subcommands for:
  - Missing required flags (e.g., `-target` for proxy)
  - Invalid IP/port parsing
  - Network listen/bind failures
  - Database open/close errors (record/replay)
  - Flow generation failures (barrage)
  - Any unrecoverable runtime error logged via `log.Fatal` or `log.Fatalf`
- **Exit 2** is exclusive to `main.go` when an unrecognized subcommand is passed (e.g., `flowgre foobar`). Valid subcommands are: `single`, `barrage`, `ipfix`, `record`, `replay`, `proxy`, `version`, `help`.

Signal handlers (`SIGINT`, `SIGTERM`) trigger graceful shutdown and exit with code `0`.

## Configuration Keys Reference

When using `flowgre barrage -config <file.yaml>`, the YAML config supersedes all command-line flags. Config is loaded via [Viper](https://github.com/spf13/viper) from the `config` package ([`config/config.go`](config/config.go)).

Only **one target** is allowed per config file. The target name is arbitrary.

### YAML Schema

```yaml
targets:
  <name>:                          # Arbitrary target name (only one allowed)
    ip: "127.0.0.1"               # Collector IP address
    port: 9995                    # Collector UDP port
    workers: 4                    # Concurrent workers
    delay: 100                    # Milliseconds between packets
    template-interval: 30         # Seconds between template retransmissions (0 = disable)
    src-range: "10.0.0.0/8"      # CIDR range for source IPs
    dst-range: "10.0.0.0/8"      # CIDR range for destination IPs
    web: false                    # Enable web dashboard
    web-ip: "127.0.0.1"          # Web server listen address (default: loopback)
    web-port: 8080               # Web server port
    web-username: ""             # Web server username (or use FLOWGRE_WEB_USERNAME env var)
    web-password: ""             # Web server password (or use FLOWGRE_WEB_PASSWORD env var)
    protocol: "netflow"          # Protocol: "netflow" or "ipfix"
```

### Key Descriptions

| Key | Type | Default | CLI Equivalent | Description |
|---|---|---|---|---|
| `ip` | string | `127.0.0.1` | `-server` | Collector hostname or IP address (IPv4/IPv6) |
| `port` | int | `9995` | `-port` | Collector UDP port |
| `workers` | int | `4` | `-workers` | Number of concurrent sender workers |
| `delay` | int | `100` | `-delay` | Milliseconds between packets per worker |
| `template-interval` | int | `30` | `-template-interval` | Seconds between NetFlow/IPFIX template retransmissions. Set to `0` to disable retransmission |
| `src-range` | string | `10.0.0.0/8` | `-src-range` | CIDR notation for source IP pool (auto-detects IPv4 vs IPv6) |
| `dst-range` | string | `10.0.0.0/8` | `-dst-range` | CIDR notation for destination IP pool (auto-detects IPv4 vs IPv6) |
| `web` | bool | `false` | `-web` | Enable the built-in web dashboard |
| `web-ip` | string | `127.0.0.1` | `-web-ip` | Bind address for the web dashboard (IPv4/IPv6). Defaults to loopback for safety |
| `web-port` | int | `8080` | `-web-port` | Listening port for the web dashboard |
| `web-username` | string | *(empty)* | `-web-username` | Web dashboard username. Falls back to `FLOWGRE_WEB_USERNAME` env var. Defaults to `admin` |
| `web-password` | string | *(empty)* | `-web-password` | Web dashboard password. Falls back to `FLOWGRE_WEB_PASSWORD` env var. If omitted, a random password is generated and printed at startup |
| `protocol` | string | `netflow` | `-protocol` | Export protocol: `netflow` (NetFlow v9) or `ipfix` (IPFIX/RFC 7011) |

Note: The `profile` flag (`-profile`) has **no config file equivalent** — it is only available via the CLI for the `barrage` subcommand and controls the NetFlow field set (`generic`, `minimal`, `extended`).

Note: The `updatets` flag (`-updatets`) has **no config file equivalent** — it is only available via the CLI for the `replay` subcommand.

Note: When binding the web dashboard to a non-loopback address (e.g., `0.0.0.0`), explicit credentials are required via CLI flags, YAML config, or environment variables. Startup will fail otherwise.

---

## Single Mode

```shell
Single is used to send a given number of flows in sequence to a collector for testing.

Usage of flowgre single:

  -count int
        count of flows to send in sequence. (default 1)
  -dst-range string
        CIDR range to use for generating destination IPs for flows (default "10.0.0.0/8")
  -hexdump
        If true, do a hexdump of the packet
  -port int
        destination port used by the flow collector. (default 9995)
  -server string
        servername or IP address of flow collector. (default "127.0.0.1")
  -src-port int
        source port used by the client. If 0, a random port between 10000-15000 is used
  -src-range string
        CIDR range to use for generating source IPs for flows (default "10.0.0.0/8")
```

### Example Use
```shell
flowgre single -server 10.10.10.10 -count 10
```

### IPv6 Example

IPv6 is supported natively — just pass IPv6 CIDRs and the system auto-detects:

```shell
flowgre single -server 2001:db8::1 -src-range 2001:db8:1::/48 -dst-range 2001:db8:2::/48 -count 10
```

## Barrage Mode

```shell
Barrage is used to send a continuous barrage of flows in different sequences to a collector for testing.

Usage of flowgre barrage:

  -config string
        Config file to use. Supersedes all given args
  -delay int
        number of milliseconds between packets sent (default 100)
  -dst-range string
        CIDR range to use for generating destination IPs for flows (default "10.0.0.0/8")
  -port int
        destination port used by the flow collector (default 9995)
  -server string
        servername or IP address of the flow collector (default "127.0.0.1")
  -src-range string
        CIDR range to use for generating source IPs for flows (default "10.0.0.0/8")
  -protocol string
        protocol to use: netflow or ipfix (default "netflow")
  -profile string
        flow profile for netflow: generic, minimal, extended (default "generic")
  -template-interval int
        seconds between template retransmissions (default 30, 0 to disable)
  -web
        Whether to use the web server or not
  -web-ip string
        IP address the web server will listen on (default "127.0.0.1")
  -web-port int
        Port to bind the web server on (default 8080)
  -web-username string
        Web server username (default: env FLOWGRE_WEB_USERNAME or "admin")
  -web-password string
        Web server password (default: env FLOWGRE_WEB_PASSWORD or generated)
  -workers int
        number of workers to create. Unique sources per worker (default 4)
```

## Example Config File

```yaml
targets:
  server1:
    ip: 127.0.0.1
    port: 9995
    workers: 4
    delay: 100
```

## IPFIX Mode

IPFIX (IP Flow Information Export, RFC 7011) is the IETF standard successor to NetFlow v9. Flowgre generates IPFIX export packets using IANA-defined field type numbers for compatibility with standard IPFIX collectors.

### Single IPFIX Mode

Send a given number of IPFIX flows in sequence to a collector for testing.

```shell
IPFIX is used to send a given number of IPFIX flows in sequence to a collector for testing.

Usage of flowgre ipfix:

  -count int
        count of flows to send in sequence. (default 1)
  -dst-range string
        CIDR range to use for generating destination IPs for flows (default "10.0.0.0/8")
  -hexdump
        If true, do a hexdump of the packet
  -port int
        destination port used by the flow collector. (default 9995)
  -server string
        servername or IP address of flow collector. (default "127.0.0.1")
  -src-port int
        source port used by the client. If 0, a random port between 10000-15000 is used
  -src-range string
        CIDR range to use for generating source IPs for flows (default "10.0.0.0/8")
```

### Example Use
```shell
flowgre ipfix -server 10.10.10.10 -count 10
```

### IPFIX Barrage Mode

Send a continuous barrage of IPFIX flows to a collector by using `--protocol ipfix` with the barrage subcommand:

```shell
flowgre barrage -server 10.10.10.10 -protocol ipfix -workers 4 -delay 100
```

The IPFIX field types used follow the [IANA IPFIX Information Model](https://www.iana.org/assignments/ipfix/ipfix.xhtml):

| IPFIX Field Type | Value | Description |
|---|---|---|
| octetDeltaCount | 1 | Input bytes |
| postOctetDeltaCount | 23 | Output bytes |
| packetDeltaCount | 2 | Input packets |
| postPacketDeltaCount | 24 | Output packets |
| sourceIPv4Address | 8 | Source IPv4 address |
| destinationIPv4Address | 12 | Destination IPv4 address |
| sourceIPv6Address | 27 | Source IPv6 address |
| destinationIPv6Address | 28 | Destination IPv6 address |
| sourceIPv6PrefixLength | 29 | Source IPv6 prefix length |
| destinationIPv6PrefixLength | 30 | Destination IPv6 prefix length |
| sourceTransportPort | 7 | Source port |
| destinationTransportPort | 11 | Destination port |
| protocolIdentifier | 4 | IP protocol number |
| tcpControlBits | 6 | TCP flags |
| flowStartMilliseconds | 152 | Flow start time |
| flowEndMilliseconds | 153 | Flow end time |
| flowDirection | 61 | Flow direction |
| ipClassOfService | 5 | IP ToS/CoS value |
| flowEndReason | 136 | Flow end reason |

## Record Mode

```shell
Record is used to record flows to a file for later replay testing.

Usage of flowgre record:

  -db string
        Directory to place recorded flows for later replay (default "recorded_flows")
  -ip string
        IP address record should listen on (default "127.0.0.1")
  -port int
        listen UDP port (default 9995)
  -verbose
        Whether to log every packet received. Warning: can be a lot of output
```

Record accepts both NetFlow v9 and IPFIX v10 packets and stores them in the database.

## Replay Mode

```shell
Replay is used to send recorded flows to a target server.

Usage of flowgre replay:

  -db string
        Directory to read recorded flows from (default "recorded_flows")
  -delay int
        number of milliseconds between packets sent (default 100)
  -loop
        Loops the replays forever
  -port int
        target server UDP port (default 9995)
  -server string
        target server to replay flows at (default "127.0.0.1")
  -verbose
        Whether to log every packet received. Warning: can be a lot of output
  -workers int
        Number of workers to spawn for replay (default 1)
```

## Proxy Mode

```shell
Proxy is used to accept flows and relay them to multiple targets.

Usage of flowgre proxy:

  -ip string
        IP address proxy should listen on (default "127.0.0.1")
  -port int
        proxy listen UDP port (default 9995)
  -target value
        Can be passed multiple times in IP:PORT format
  -verbose
        Whether to log every flow received. Warning: can be a lot of output
```

## Web Dashboard

Flowgre provides a basic web dashboard that will display the number of workers, how much work they've done and the config used to start Flowgre. The stats shown all come from the stats collector and should match the stdout worker stats.

The web dashboard defaults to binding on `127.0.0.1` (loopback) for security. When binding to a non-loopback address, explicit credentials are required via CLI flags, YAML config, or environment variables (`FLOWGRE_WEB_USERNAME`/`FLOWGRE_WEB_PASSWORD`).

If no credentials are provided, a random password is generated and printed at startup. Basic Authentication should be placed behind TLS when used across an untrusted network.

![Dashboard Image](https://github.com/dmabry/flowgre/blob/main/docs/images/dashboard.png?raw=true)

## License

Licensed to the Flowgre Team under one or more contributor license agreements. The Flowgre Team licenses this file to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

Please see the [LICENSE](LICENSE) file included in the root directory of the source tree for extended license details.

## Additional Resources

### [Contributing Guidelines](CONTRIBUTING.md)
Please see our [Contributing Guidelines](CONTRIBUTING.md) for information on how to contribute to Flowgre.

### [Code of Conduct](CODE_OF_CONDUCT.md)
Please see our [Code of Conduct](CODE_OF_CONDUCT.md) for information on maintaining a positive and inclusive community.

### Repository Structure

```
flowgre/
├── main.go                    # CLI entry point, subcommand dispatch
├── cmd/                       # Per-mode command structs (single, barrage, record, replay, proxy)
├── netflow/                   # NetFlow v9 packet generation library
│   ├── session.go             # Session struct (replaces global state)
│   ├── flow.go                # GenericFlow, port/proto constants
│   ├── template.go            # Header, Field, Template, TemplateFlowSet
│   ├── dataflowset.go         # DataFlowSet, DataItem
│   └── packet.go              # Netflow struct + ToBytes serialization
├── ipfix/                     # IPFIX (RFC 7011) packet generation library
│   ├── ipfix.go               # Header, Field, Template, GenericFlow, DataFlowSet, IPFIX struct
│   └── single.go              # IPFIX single-mode placeholder
├── lifecycle/                 # Shared process management (context, signals, WaitGroup)
├── config/                    # Viper-based YAML configuration loading
├── stats/                     # Worker statistics collection
├── models/                    # Pure data structures (no concurrency primitives)
├── utils/                     # Focused utilities (rand, ip, packet)
│   ├── rand.go                # Random number generation
│   ├── ip.go                  # IP math and CIDR operations
│   ├── packet.go              # Packet sending
│   └── utils.go               # Binary encoding helpers
├── web/                       # Web dashboard for barrage monitoring
├── barrage/                   # Barrage mode implementation (NetFlow + IPFIX)
├── single/                    # Single mode implementation
├── record/                    # Record mode implementation
├── replay/                    # Replay mode implementation
├── proxy/                     # Proxy mode implementation
└── ...                        # Config files, docs, etc.
```

## Architecture

Flowgre uses a **command pattern** for CLI dispatch: each subcommand (`single`, `barrage`, `record`, `replay`, `proxy`) has its own struct in `cmd/` with `ParseFlags()` and `Execute()` methods. The main entry point (`main.go`) routes to the appropriate command.

NetFlow v9 generation uses a **Session-based** design — each invocation creates a fresh `netflow.Session` instead of relying on package-level globals, making the library thread-safe and testable.

All modes share a common **lifecycle manager** (`lifecycle/`) that handles context creation, signal handling (SIGINT/SIGTERM), and WaitGroup coordination, eliminating duplicated boilerplate across modes.

## Development Practices

### Code Structure

- Follow Go best practices from [Effective Go](https://go.dev/doc/effective_go)
- Use `context.Context` for cancellation propagation

### Testing

- Write tests in `_test.go` files
- Run tests with race detector: `go test -race ./...`

### Dependency Management

- Keep dependencies minimal
- Prefer standard library packages when possible

## Pull Request Template

Use this template when creating PRs:

```
## Description
Brief summary of changes made.

## Related Issue
Closes #<issue-number>

## Checklist
- [ ] Code is properly formatted
- [ ] Linting passes
- [ ] Tests added/updated
- [ ] Documentation updated
```