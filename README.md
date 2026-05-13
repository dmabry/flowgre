
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

For sending fabricated NetFlow v9 and IPFIX (RFC 7011) traffic to a collector for testing

[![Go Tests](https://github.com/dmabry/flowgre/actions/workflows/go-test.yml/badge.svg)](https://github.com/dmabry/flowgre/actions/workflows/go-test.yml)
[![Security Scan](https://github.com/dmabry/flowgre/actions/workflows/security.yml/badge.svg)](https://github.com/dmabry/flowgre/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmabry/flowgre)](https://goreportcard.com/report/github.com/dmabry/flowgre)
[![Go Reference](https://pkg.go.dev/badge/github.com/dmabry/flowgre.svg)](https://pkg.go.dev/github.com/dmabry/flowgre)

## Table of Contents
- [Single Mode](#single-mode)
- [Barrage Mode](#barrage-mode)
- [IPFIX Mode](#ipfix-mode)
- [Record Mode](#record-mode)
- [Replay Mode](#replay-mode)
- [Proxy Mode](#proxy-mode)
- [Web Dashboard](#web-dashboard)
- [License](#license)

## Build / Lint / Test Commands
- `go build` – compile the binary.
- `go test -v ./... -count 1` – run all tests once, verbose output.
- `go test -race ./...` – run tests with race detector.
- `GOEXPERIMENT=goroutineleakprofile go test ./...` – verify no goroutine leaks (Go 1.26+).
- `golangci-lint run` – linting (requires golangci‑lint installed).
- `gosec ./...` – security scan.
- `trivy fs .` – container image scanning.

## Code Style Guidelines
- Imports sorted alphabetically; group standard library first.
- Use `context.Context` for cancellation and timeouts.
- Exported names: PascalCase; unexported: lowerCamelCase.
- Constants: UpperCamelCase or UPPERCASE if global.
- Errors returned as `fmt.Errorf("...")`; wrap errors with context.
- Use `go vet`, `staticcheck`, `golangci-lint` for static analysis.
- Formatting: run `gofmt -w .` before commit.
- Documentation: comment every exported function and type.

## CI / GitHub Actions
The project uses the following GitHub actions:
- **Go Tests** (`.github/workflows/go-test.yml`) – runs tests on push to main and pull requests.
  - Steps include checkout, Go setup, `go mod tidy`, test run (`go test -v ./... -count 1`), and build.
- **Auto‑Merge Dependabot** (`.github/workflows/auto-merge-dependabot.yml`) – merges Dependabot PRs automatically.
- **Release** (`.github/workflows/release.yml`) – builds release artifacts for multiple platforms.
  - Triggered on tag pushes matching `v*` pattern.
  - Automatically updates nfpm configs at runtime using `sed`.
  - Creates packages for RPM, DEB, APK, and standalone binaries.
  - Generates SBOM using Trivy v0.70.0.
- **Security Scan** (`.github/workflows/security.yml`) – runs `trivy` to scan images.

Ensure that any new CI configuration follows the pattern above and includes relevant steps.

## Environment Variables & Configuration
The project uses Viper for configuration handling (`https://github.com/spf13/viper`).
Command‑line arguments, YAML config files, and environment variables are supported.
If you need to supply environment variables, create a `.env` file in the repository root (e.g., `export FLOWGRE_DEBUG=1`) or set them directly when running commands.

Dependencies which read environment variables include `github.com/subosito/gotenv`. Use `gotenv.Load()` in your code as needed.

## Go Version Requirement
- **Required:** Go 1.26 or later (latest stable).
- Install from https://go.dev/dl/ or use `go env -w GOTOOLCHAIN=auto`.
- Experimental features available: `goroutineleakprofile`, `simd`, `runtimesecret`.

## Dependency Management
- Keep dependencies minimal and use standard library packages where possible.
- Run `go mod tidy` before building/testing to remove unused modules.
- Use `go get -u` for updating dependencies when necessary, but always check the changelog.
- **Security:** Address Dependabot alerts promptly by updating vulnerable dependencies.
- Current known issues: None (all alerts resolved as of v0.5.14).

## Testing Practices
- Write tests in `_test.go` files.
- Run tests with race detector: `go test -race ./...`.
- Ensure coverage is adequate; run `go test -coverprofile=coverage.out` and generate reports if required.
- Include integration tests where applicable (e.g., `netflow_test.go`, `record_test.go`, etc.).
- All tests must pass before merging PRs or creating releases.

## Branching Strategy

### Main Branch
- `main` is the primary development branch.
- All features should be developed in feature branches off `main`.
- PRs must pass all CI checks before merging.

### Feature Branches
- Naming: `feature/<description>` or `fix/<description>`
- Created from: `main`
- Target: Merge back to `main` via PR
- Example: `git checkout -b feature/my-feature main`

### Release Branches
- Naming: `release/<major>.<minor>` (e.g., `release/0.5`)
- Created from: `main` when starting a new release cycle
- Purpose: Long-lived branch for patch releases (e.g., v0.5.1, v0.5.2)
- Hotfixes can be applied to this branch and merged back to `main`
- Example: `git checkout -b release/0.5 main`

### Release Tags
- Naming: `v<major>.<minor>.<patch>` (e.g., `v0.5.15`)
- Created from: Release branch (e.g., `release/0.5`)
- Pushing a tag triggers the Release workflow automatically
- Example:
  ```bash
  git checkout release/0.5
  git tag v0.5.15
  git push origin v0.5.15
  ```

## PR Template / Commit Messages
Use the provided pull request template in `.github/ISSUE_TEMPLATE/feature_request.md`. Commit messages should be concise, describing *why* the change was made:
```
Add: [Feature] – brief description of new feature
Fix: [Bug] – description of bug fixed
Update: [Enhancement] – details of improvement
```

## Release Workflow Details
The Release workflow (`.github/workflows/release.yml`) performs the following steps:
1. **Checkout code** – Pull the repository at the tagged commit.
2. **Set up Go** – Install Go 1.26.
3. **Install build dependencies** – Install build-essential.
4. **Build multi-platform** – Run `scripts/build-multiplatform.sh` with the version.
5. **List built files** – Verify all binaries were created.
6. **Update nfpm configs** – Dynamically update version and binary paths using `sed`.
7. **Build musl version** – Build Alpine-compatible binary.
8. **Install nfpm** – Install package manager tool.
9. **Package RPM and DEB** – Create Linux packages.
10. **Package APK** – Create Alpine package.
11. **Generate SBOM** – Create Software Bill of Materials using Trivy.
12. **Create GitHub Release** – Upload all artifacts to GitHub.

**Important:** The nfpm config update step is critical. It ensures the config files match the actual binary names generated during the build.

### Common Workflows

### Creating a Feature Branch
```bash
git checkout main
git pull origin main
git checkout -b feature/my-feature
# Make changes, commit, push
git push -u origin feature/my-feature
```

### Creating a Release
```bash
# Create release branch if not exists
git checkout main
git pull origin main
git checkout -b release/0.5  # Or use existing branch

# Create and push tag
git tag v0.5.15
git push origin v0.5.15

# Monitor CI at https://github.com/dmabry/flowgre/actions
```

### Hotfixing a Release
```bash
# Make changes on release branch
git checkout release/0.5
# Fix issue, commit
git commit -m "Fix: Critical bug fix"

# Create new patch tag
git tag v0.5.16
git push origin v0.5.16

# Merge fix back to main
git checkout main
git merge release/0.5
git push origin main
```

### Addressing Dependabot Alerts
```bash
# Create branch for security fix
git checkout -b security/fix-vulnerability main

# Update vulnerable dependency
go get <package>@<new-version>
go mod tidy

# Test thoroughly
go test -v ./... -count 1

# Create PR and merge
git push -u origin security/fix-vulnerability
# Create PR on GitHub
```

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
  -web
        Whether to use the web server or not
  -web-ip string
        IP address the web server will listen on (default "0.0.0.0")
  -web-port int
        Port to bind the web server on (default 8080)
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
| inOctets | 1026 | Input bytes |
| outOctets | 1028 | Output bytes |
| inPackets | 1025 | Input packets |
| outPackets | 1027 | Output packets |
| sourceIPv4Address | 8 | Source IPv4 address |
| destinationIPv4Address | 12 | Destination IPv4 address |
| sourceTransportPort | 7 | Source port |
| destinationTransportPort | 11 | Destination port |
| protocolIdentifier | 4 | IP protocol number |
| tcpFlags | 6 | TCP flags |
| flowStartMilliseconds | 152 | Flow start time |
| flowEndMilliseconds | 153 | Flow end time |
| flowDirection | 1024 | Flow direction |
| ipClassOfService | 3 | IP ToS/CoS value |

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
