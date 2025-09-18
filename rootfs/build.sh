#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] ✅${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠️${NC}  $1"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ❌${NC} $1"
    exit 1
}

usage() {
    echo "Usage: $0 [OPTIONS] <ARCHITECTURE>"
    echo ""
    echo "Build Viper microVM images using Packer"
    echo ""
    echo "ARCHITECTURES:"
    echo "  aarch64    Build ARM64 image (for Apple Silicon development)"
    echo "  x86_64     Build x86_64 image (for production deployment)"
    echo ""
    echo "OPTIONS:"
    echo "  -h, --help     Show this help message"
    echo "  -v, --verbose  Enable verbose output"
    echo "  -c, --clean    Clean build artifacts before building"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 x86_64                    # Build production x86_64 image"
    echo "  $0 aarch64                   # Build development ARM64 image"
    echo "  $0 --clean x86_64            # Clean and build x86_64 image"
    echo ""
}

build_agent_binary() {
    log "Building viper-agent for $ARCH..."

    cd "$PROJECT_ROOT"

    case "$ARCH" in
        x86_64)
            # Build x86_64 Linux binary for production
            GOOS=linux GOARCH=amd64 go build \
                -trimpath \
                -ldflags "-w -s" \
                -o "$PROJECT_ROOT/bin/viper-agent-linux-amd64" \
                ./cmd/agent

            # Create symlink for Packer template
            ln -sf viper-agent-linux-amd64 "$PROJECT_ROOT/bin/viper-agent"
            success "x86_64 viper-agent built"
            ;;
        aarch64)
            # Build ARM64 binary (native or cross-compile)
            go build \
                -trimpath \
                -ldflags "-w -s" \
                -o "$PROJECT_ROOT/bin/viper-agent" \
                ./cmd/agent
            success "ARM64 viper-agent built"
            ;;
    esac
}

check_requirements() {
    log "Checking build requirements..."

    # Check Packer
    if ! command -v packer >/dev/null 2>&1; then
        error "Packer is not installed. Please install Packer first."
    fi

    # Check Go (needed to build viper-agent)
    if ! command -v go >/dev/null 2>&1; then
        error "Go is not installed. Please install Go first."
    fi

    # Check architecture-specific requirements
    case "$ARCH" in
        x86_64)
            if ! command -v qemu-system-x86_64 >/dev/null 2>&1; then
                error "qemu-system-x86_64 not found. Please install QEMU."
            fi
            ;;
        aarch64)
            if ! command -v qemu-system-aarch64 >/dev/null 2>&1; then
                error "qemu-system-aarch64 not found. Please install QEMU."
            fi
            ;;
    esac

    success "All requirements satisfied"
}

clean_artifacts() {
    log "Cleaning build artifacts..."

    rm -rf "$SCRIPT_DIR"/output-*
    rm -rf "$SCRIPT_DIR"/out/
    rm -f "$SCRIPT_DIR"/*.qcow2
    rm -f "$SCRIPT_DIR"/*.img
    rm -f "$SCRIPT_DIR"/packer_cache/*

    success "Build artifacts cleaned"
}

build_image() {
    local template_file=""
    local build_name=""

    case "$ARCH" in
        x86_64)
            template_file="alpine-x86_64.pkr.hcl"
            build_name="viper-alpine-x86_64-rootfs"
            ;;
        aarch64)
            template_file="alpine.pkr.hcl"
            build_name="viper-alpine-rootfs"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    if [ ! -f "$SCRIPT_DIR/$template_file" ]; then
        error "Template file not found: $template_file"
    fi

    log "Building $ARCH microVM image using $template_file..."

    cd "$SCRIPT_DIR"

    if [ "$VERBOSE" = true ]; then
        PACKER_LOG=1 packer build "$template_file"
    else
        packer build "$template_file"
    fi

    success "$ARCH microVM image built successfully"

    # Show build results
    log "Build results:"
    find "$SCRIPT_DIR" -name "*.qcow2" -o -name "viper-vm*" -type f | head -5 | while read -r file; do
        size=$(du -h "$file" | cut -f1)
        echo "  📦 $(basename "$file") ($size)"
    done
}

# Parse command line arguments
VERBOSE=false
CLEAN=false
ARCH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        aarch64|x86_64)
            ARCH=$1
            shift
            ;;
        *)
            error "Unknown argument: $1"
            ;;
    esac
done

# Validate architecture argument
if [ -z "$ARCH" ]; then
    error "Architecture argument required. Use --help for usage."
fi

# Main execution
log "🐍 Viper microVM Image Builder"
log "Architecture: $ARCH"

if [ "$CLEAN" = true ]; then
    clean_artifacts
fi

check_requirements
build_agent_binary
build_image

success "🎉 Build completed successfully!"
log "Your $ARCH microVM image is ready for deployment."