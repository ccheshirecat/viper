#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/dist"

# Configuration
IMAGE_NAME="viper-headless"
CONTAINER_NAME="viper-export-$$"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[BUILD]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Verify viper-agent binary exists
if [ ! -f "$PROJECT_ROOT/bin/viper-agent" ]; then
    error "viper-agent binary not found at $PROJECT_ROOT/bin/viper-agent. Run 'make build-agent' first."
fi

log "Building Docker image with viper-agent..."
cd "$PROJECT_ROOT"
docker build -t "$IMAGE_NAME:latest" -f images/headless/Dockerfile .

log "Creating container for export..."
docker create --name "$CONTAINER_NAME" "$IMAGE_NAME:latest" /bin/true

# Cleanup function
cleanup() {
    log "Cleaning up container..."
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

log "Exporting container filesystem..."
docker export "$CONTAINER_NAME" > "$OUTPUT_DIR/viper-rootfs.tar"

log "Converting tar to disk image..."
# Create a disk image from the tar export
DISK_SIZE="2G"
DISK_IMAGE="$OUTPUT_DIR/viper-headless.img"

# Create raw disk image
dd if=/dev/zero of="$DISK_IMAGE" bs=1M count=0 seek=2048 2>/dev/null

# Create filesystem
mkfs.ext4 -F "$DISK_IMAGE" >/dev/null 2>&1

# Mount and extract
MOUNT_DIR=$(mktemp -d)
cleanup_mount() {
    umount "$MOUNT_DIR" 2>/dev/null || true
    rmdir "$MOUNT_DIR" 2>/dev/null || true
    cleanup
}
trap cleanup_mount EXIT

log "Mounting disk image and extracting filesystem..."
sudo mount -o loop "$DISK_IMAGE" "$MOUNT_DIR"
sudo tar -xf "$OUTPUT_DIR/viper-rootfs.tar" -C "$MOUNT_DIR"

# Ensure agent is executable and properly configured
sudo chmod +x "$MOUNT_DIR/usr/local/bin/viper-agent"
sudo chmod +x "$MOUNT_DIR/init"

# Create qcow2 version for smaller size
log "Creating qcow2 image..."
QCOW2_IMAGE="$OUTPUT_DIR/viper-headless.qcow2"
sudo umount "$MOUNT_DIR"
qemu-img convert -f raw -O qcow2 -c "$DISK_IMAGE" "$QCOW2_IMAGE"

# Get file sizes
RAW_SIZE=$(du -h "$DISK_IMAGE" | cut -f1)
QCOW2_SIZE=$(du -h "$QCOW2_IMAGE" | cut -f1)

log "Build complete!"
log "Raw image:  $DISK_IMAGE ($RAW_SIZE)"
log "QCOW2 image: $QCOW2_IMAGE ($QCOW2_SIZE)"
log "Use the QCOW2 image with your Cloud Hypervisor Nomad driver"

# Generate job template example
cat > "$OUTPUT_DIR/example-job.hcl" << 'EOF'
job "viper-test" {
  datacenters = ["dc1"]
  type        = "service"

  group "browser" {
    count = 1

    task "viper-vm" {
      driver = "virt"

      config {
        image = "/path/to/viper-headless.qcow2"
        hostname = "viper-browser"

        # Boot directly to our agent
        cmdline = "console=ttyS0 init=/init"

        # Network configuration
        network_interface {
          bridge {
            name = "br0"
            # static_ip will be assigned automatically or via DHCP
          }
        }
      }

      resources {
        cpu    = 1000  # 1 CPU core
        memory = 1024  # 1GB RAM
      }

      # Health check for the agent
      service {
        name = "viper-agent"
        port = "http"

        check {
          type     = "tcp"
          port     = "http"
          interval = "30s"
          timeout  = "5s"
        }
      }
    }

    network {
      port "http" {
        to = 8080
      }
    }
  }
}
EOF

log "Example Nomad job template created at: $OUTPUT_DIR/example-job.hcl"