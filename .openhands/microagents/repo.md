# flowgre Repository Guide

## General Setup
This repository contains the flowgre project, a Go-based application for network traffic analysis. To set up the development environment:

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
├── cmd/                # Main application entrypoints
├── internal/             # Private application packages
│   ├── config/           # Configuration management
│   ├── collector/        # Network data collection components
│   └── processor/        # Data processing logic
├── pkg/                  # Shared public packages
├── proto/                # Protocol buffer definitions
├── tests/                # Integration and unit tests
└── go.mod                # Go module definition
```

## Development Practices
### Code Structure
- Use dependency injection with Wire (https://github.com/google/wire)
- Follow Go best practices from https://go.dev/doc/effective_go
- Use context.Context for cancellation propagation

### Testing
- Write tests in `_test.go` files
- Use testify for assertions: `github.com/stretchr/testify`
- Run tests with race detector: `go test -race ./...`
- Add benchmarks for performance-critical code

### Dependency Management
- Keep dependencies minimal
- Prefer standard library packages when possible
- Use module versioning for public packages

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
- [ ] CHANGELOG.md entry added
```

## Implementation Details
### Configuration Management
- Configuration is handled through viper: https://github.com/spf13/viper
- Environment variables follow the format FLOWGRE_<SECTION>_<KEY>
- Default values are set in internal/config/config.go

### Metrics & Monitoring
- Prometheus metrics exposed at /metrics
- Use standard Go metrics from expvar
- Add new metrics in internal/monitoring/metrics.go

### Security Practices
- All network input is validated
- Use context timeouts for all operations
- Regular security audits of dependencies