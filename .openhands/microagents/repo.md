
# flowgre Repository Guide

## General Setup
This repository contains the flowgre project, a Go-based tool designed for generating and testing Netflow traffic. To set up the development environment:

1. Install Go 1.21+ and set up your GOPROXY
2. Install dependencies: `go mod download`
3. Install development tools:
   ```bash
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

## Code Quality Requirements
Before submitting changes, ensure:
1. Code is formatted with `go fmt ./...`
2. Imports are organized with `goimports -w .`
3. Linting passes: `golangci-lint run --config .golangci.yml`
4. All tests pass: `go test -race ./...`

## Repository Structure
```
flowgre/
├── barrage/             # Barrage mode implementation for continuous traffic generation
├── examples/            # Example configurations and usage scenarios
├── flow/                # Core flow generation logic
├── models/              # Data models used across the application
├── proxy/               # Proxy functionality to relay flows to multiple targets
├── record/              # Record mode implementation for capturing Netflow traffic
├── replay/              # Replay mode implementation for sending recorded traffic
├── scripts/             # Utility scripts for development and testing
├── single/              # Single mode implementation for sequential flow generation
├── utils/               # Helper functions and utilities
├── web/                 # Web dashboard for monitoring Flowgre activity
├── .github/             # GitHub Actions workflows and configurations
├── .nfpm/               # NFPM configuration for packaging
├── .openhands/          # OpenHands-specific configurations
├── CODE_OF_CONDUCT.md   # Code of Conduct guidelines
├── CONTRIBUTING.md      # Contribution guidelines
├── Dockerfile           # Docker container definition
├── LICENSE              # License information (Apache 2.0)
├── README.md            # Project overview and usage instructions
├── flowgre.go           # Main entry point for the application
├── go.mod               # Go module dependencies
├── go.sum               # Go module dependency versions
```

## Development Practices
### Code Structure
- Follow Go best practices from https://go.dev/doc/effective_go
- Use context.Context for cancellation propagation

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

## Implementation Details
### Configuration Management
- Configuration is handled through viper: https://github.com/spf13/viper
- Command-line arguments and YAML config files are supported

### Modes of Operation
Flowgre supports multiple modes for generating and testing Netflow traffic:
1. **Single**: Sends a specified number of flows in sequence to a collector.
2. **Barrage**: Sends a continuous barrage of flows, useful for stress testing collectors.
3. **Record**: Records incoming Netflow traffic to files for later replay.
4. **Replay**: Replays previously recorded Netflow traffic to a target server.
5. **Proxy**: Accepts Netflow traffic and relays it to multiple targets.

### Web Dashboard
Flowgre includes a web dashboard that displays:
- Number of workers
- Work completed by each worker
- Configuration used to start Flowgre

## Security Practices
- All network input is validated
- Use context timeouts for all operations
