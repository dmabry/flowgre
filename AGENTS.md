# AGENTS

## Build / Lint / Test Commands
- `go build` – compile the binary.
- `go test -v ./... -count 1` – run all tests once, verbose output.
- `golangci-lint run` – linting (requires golangci‑lint installed).
- `gosec ./...` – security scan.
- `trivy fs .` – container image scanning.

## Code Style Guidelines
- Imports sorted alphabetically; group standard library first.
- Use `context.Context` for cancellation and timeouts.
- Exported names: PascalCase; unexported: lowerCamelCase.
- Constants: UpperCamelCase or UPPERCASE if global.
- Errors returned as `fmt.Errorf("...")`; wrap errors with context.
- Use `go vet`, `staticcheck`, `golangci‑lint` for static analysis.
- Formatting: run `gofmt -w .` before commit.
- Documentation: comment every exported function and type.

## CI / GitHub Actions
The project uses the following GitHub actions:
- **Go Tests** (`.github/workflows/go-test.yml`) – runs tests on push to main and pull requests.
  - Steps include checkout, Go setup, `go mod tidy`, test run (`go test -v ./... -count 1`), and build.
- **Auto‑Merge Dependabot** (`.github/workflows/auto-merge-dependabot.yml`) – merges Dependabot PRs automatically.
- **Release** (`.github/workflows/release.yml`) – builds release artifacts for multiple platforms.
- **Security Scan** (`.github/workflows/security.yml`) – runs `trivy` to scan images.

Ensure that any new CI configuration follows the pattern above and includes relevant steps.

## Environment Variables & Configuration
The project uses Viper for configuration handling (`https://github.com/spf13/viper`).
Command‑line arguments, YAML config files, and environment variables are supported.
If you need to supply environment variables, create a `.env` file in the repository root (e.g., `export FLOWGRE_DEBUG=1`) or set them directly when running commands.

Dependencies that read environment variables include `github.com/subosito/gotenv`. Use `gotenv.Load()` in your code as needed.

## Dependency Management
- Keep dependencies minimal and use standard library packages where possible.
- Run `go mod tidy` before building/testing to remove unused modules.
- Use `go get -u` for updating dependencies when necessary, but always check the changelog.

## Testing Practices
- Write tests in `_test.go` files.
- Run tests with race detector: `go test -race ./...`.
- Ensure coverage is adequate; run `go test -coverprofile=coverage.out` and generate reports if required.
- Include integration tests where applicable (e.g., `netflow_test.go`, `record_test.go`, etc.).

## PR Template / Commit Messages
Use the provided pull request template in `.github/ISSUE_TEMPLATE/feature_request.md`. Commit messages should be concise, describing *why* the change was made:
```
Add: [Feature] – brief description of new feature
Fix: [Bug] – description of bug fixed
Update: [Enhancement] – details of improvement
```

## Cursor / Copilot Rules
No cursor or copilot rules found in this repo. If added, include them here.
