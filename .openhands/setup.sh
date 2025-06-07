
#!/bin/bash
set -e

# Function to print messages with timestamp
log() {
  echo "$(date +'%Y-%m-%d %H:%M:%S') - $1"
}

# Install Go (if not already installed)
log "Checking for Go installation..."
if ! which go &>/dev/null; then
  log "Go is not installed. Downloading and installing the latest version..."

  # Get the latest Go version
  GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | grep go)

  # Download and install Go
  wget -q "https://golang.org/dl/${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/golang.tar.gz
  sudo tar -C /usr/local -xzf /tmp/golang.tar.gz

  # Set up environment variables
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
  echo 'export PATH=$PATH:~/.go/bin' >> ~/.profile
  source ~/.profile
  export PATH=${PATH}:/usr/local/go:/usr/local/go/bin

  log "Go ${GO_VERSION} installed successfully."
else
  GO_VERSION=$(go version | awk '{print $3}' | sed 's/,//')
  log "Go is already installed (version: ${GO_VERSION})."
fi

# Set up GOPROXY if not set
log "Configuring GOPROXY..."
if [ -z "$GOPROXY" ]; then
  export GOPROXY=https://proxy.golang.org
  echo 'export GOPROXY=https://proxy.golang.org' >> ~/.profile
fi

# Download Go module dependencies
log "Downloading Go module dependencies..."
cd /workspace/flowgre
go mod download

# Install development tools
log "Installing development tools..."
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

log "Development environment setup completed successfully."
