# AGENTS

## Build / Lint / Test Commands
- `go build` – compile the binary.
- `gofmt -l .` – list files with formatting issues (run after every code change).
- `gofmt -w .` – auto-format all Go files.
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
- Use `go vet`, `staticcheck`, `golangci‑lint` for static analysis.
- Formatting: run `gofmt -l .` after every code change to verify formatting, then `gofmt -w .` to fix any issues.
- Documentation: comment every exported function and type.

## CI / GitHub Actions
The project uses the following GitHub actions:
- **Go Tests** (`.github/workflows/go-test.yml`) – runs tests on push to main and pull requests.
  - Steps include checkout, Go setup, `go mod tidy`, test run (`go test -v ./... -count 1`), and build.
  - Uses Go 1.26 with `actions/setup-go@v5`.
  - Includes a Docker service for container-based tests.
- **Auto‑Merge Dependabot** (`.github/workflows/auto-merge-dependabot.yml`) – merges Dependabot PRs automatically.
- **Release** (`.github/workflows/release.yml`) – builds release artifacts for multiple platforms.
  - Triggered on tag pushes matching `v*` pattern.
  - Uses Go 1.26 with `actions/setup-go@v5`.
  - Injects version string via `-ldflags -X main.version` for accurate version reporting.
  - Automatically updates nfpm configs at runtime using `sed`.
  - Creates packages for RPM, DEB, APK, and standalone binaries.
  - Generates SBOM using Trivy v0.70.0.
- **Security Scan** (`.github/workflows/security.yml`) – runs `trivy` to scan images.

Ensure that any new CI configuration follows the pattern above and includes relevant steps.

## Environment Variables & Configuration
The project uses Viper for configuration handling (`https://github.com/spf13/viper`).
Command‑line arguments, YAML config files, and environment variables are supported.
If you need to supply environment variables, create a `.env` file in the repository root (e.g., `export FLOWGRE_DEBUG=1`) or set them directly when running commands.

Dependencies that read environment variables include `github.com/subosito/gotenv`. Use `gotenv.Load()` in your code as needed.

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
- Example: `git checkout -b feature/new-feature main`

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
   - Version is injected via `-ldflags -X main.version` so binaries report the correct version.
5. **List built files** – Verify all binaries were created.
6. **Update nfpm configs** – Dynamically update version and binary paths using `sed`.
7. **Build musl version** – Build Alpine-compatible binary (with version ldflags).
8. **Install nfpm** – Install package manager tool.
9. **Package RPM and DEB** – Create Linux packages.
10. **Package APK** – Create Alpine package.
11. **Generate SBOM** – Create Software Bill of Materials using Trivy.
12. **Create GitHub Release** – Upload all artifacts to GitHub.

**Important:** The nfpm config update step is critical. It ensures the config files match the actual binary names generated during the build.

## Common Workflows

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

## Cursor / Copilot Rules
No cursor or copilot rules found in this repo. If added, include them here.
