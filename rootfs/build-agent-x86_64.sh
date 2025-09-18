#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] ✅${NC} $1"
}

log "Building viper-agent for x86_64 Linux..."

cd "$PROJECT_ROOT"

# Build x86_64 Linux binary
GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-w -s" \
    -o "$PROJECT_ROOT/bin/viper-agent-linux-amd64" \
    ./cmd/agent

# Create symlink for Packer to find the correct binary
ln -sf viper-agent-linux-amd64 "$PROJECT_ROOT/bin/viper-agent"

success "x86_64 viper-agent built successfully"
log "Binary location: $PROJECT_ROOT/bin/viper-agent-linux-amd64"
log "Symlink created:  $PROJECT_ROOT/bin/viper-agent -> viper-agent-linux-amd64"