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

# Extract kernel and initramfs (required by your CH driver)
log "Extracting kernel and initramfs..."
TEMP_DIR=$(mktemp -d)
cleanup_temp() {
    rm -rf "$TEMP_DIR"
    cleanup
}
trap cleanup_temp EXIT

cd "$TEMP_DIR"
tar -xf "$OUTPUT_DIR/viper-rootfs.tar"

# Find kernel in container
KERNEL_FILE=""
for kernel_path in boot/vmlinuz* vmlinuz* boot/vmlinux*; do
    if [ -f "$kernel_path" ]; then
        KERNEL_FILE="$kernel_path"
        break
    fi
done

if [ -n "$KERNEL_FILE" ]; then
    cp "$KERNEL_FILE" "$OUTPUT_DIR/vmlinuz"
    log "✅ Kernel extracted: $OUTPUT_DIR/vmlinuz"
else
    # Try host kernel as fallback
    log "⚠️  No kernel in container, trying host kernel..."
    if ls /boot/vmlinuz* >/dev/null 2>&1; then
        cp $(ls /boot/vmlinuz* | head -1) "$OUTPUT_DIR/vmlinuz"
        log "✅ Host kernel copied: $OUTPUT_DIR/vmlinuz"
    else
        error "No kernel found in container or host"
    fi
fi

# Create initramfs from entire filesystem (this is your innovation!)
log "Creating initramfs from container filesystem..."
find . | cpio -o -H newc | gzip > "$OUTPUT_DIR/viper-initramfs.gz"

# Also create uncompressed version for flexibility
find . | cpio -o -H newc > "$OUTPUT_DIR/viper-initramfs.cpio"

log "✅ Initramfs created: $OUTPUT_DIR/viper-initramfs.gz"

# Optional: Still create disk image for compatibility
log "Creating optional disk image for compatibility..."
DISK_IMAGE="$OUTPUT_DIR/viper-headless.img"
dd if=/dev/zero of="$DISK_IMAGE" bs=1M count=0 seek=2048 2>/dev/null
mkfs.ext4 -F "$DISK_IMAGE" >/dev/null 2>&1

MOUNT_DIR=$(mktemp -d)
cleanup_mount() {
    umount "$MOUNT_DIR" 2>/dev/null || true
    rmdir "$MOUNT_DIR" 2>/dev/null || true
    cleanup_temp
}
trap cleanup_mount EXIT

sudo mount -o loop "$DISK_IMAGE" "$MOUNT_DIR"
sudo cp -a . "$MOUNT_DIR/"
sudo chmod +x "$MOUNT_DIR/usr/local/bin/viper-agent"
sudo chmod +x "$MOUNT_DIR/init"

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

# Generate job template examples for nomad-driver-ch
cat > "$OUTPUT_DIR/viper-private-subnet.hcl" << 'EOF'
# Viper VM with Private Subnet Networking
job "viper-browser" {
  datacenters = ["dc1"]
  type        = "service"

  group "browser" {
    count = 1

    task "viper-vm" {
      driver = "nomad-driver-ch"

      config {
        # Use our generated kernel + initramfs (required by your driver)
        kernel = "/path/to/vmlinuz"
        initramfs = "/path/to/viper-initramfs.gz"

        # Optional: disk image for persistence
        image = "/path/to/viper-headless.qcow2"

        hostname = "viper-browser"

        # Agent starts as PID 1 and handles network setup
        cmdline = "console=ttyS0 init=/usr/local/bin/viper-agent"

        # Private subnet networking - driver assigns IP from pool
        network_interface {
          bridge {
            name = "br0"
            # IP will be auto-assigned from pool (e.g., 192.168.1.100-200)
          }
        }
      }

      resources {
        cpu    = 1000  # 1 CPU core
        memory = 1024  # 1GB RAM
      }

      # Service discovery for agent
      service {
        name = "viper-agent"
        port = "agent"

        check {
          type     = "tcp"
          port     = "agent"
          interval = "30s"
          timeout  = "5s"
        }
      }
    }

    network {
      port "agent" {
        to = 8080  # Agent listens on 8080 inside VM
      }
    }
  }
}
EOF

cat > "$OUTPUT_DIR/viper-static-ip.hcl" << 'EOF'
# Viper VM with Static IP Assignment
job "viper-browser-static" {
  datacenters = ["dc1"]
  type        = "service"

  group "browser" {
    count = 1

    task "viper-vm" {
      driver = "nomad-driver-ch"

      config {
        kernel = "/path/to/vmlinuz"
        initramfs = "/path/to/viper-initramfs.gz"
        image = "/path/to/viper-headless.qcow2"

        hostname = "viper-browser-static"
        cmdline = "console=ttyS0 init=/usr/local/bin/viper-agent"

        # Static IP networking for predictable agent communication
        network_interface {
          bridge {
            name = "br0"
            static_ip = "192.168.1.150"
            gateway = "192.168.1.1"
            netmask = "24"
            dns = ["8.8.8.8", "1.1.1.1"]
          }
        }
      }

      resources {
        cpu    = 1000
        memory = 1024
      }

      service {
        name = "viper-agent-static"
        address = "192.168.1.150"
        port = 8080

        check {
          type     = "tcp"
          address  = "192.168.1.150"
          port     = 8080
          interval = "30s"
          timeout  = "5s"
        }
      }
    }
  }
}
EOF

log "✅ Nomad job templates created:"
log "  - Private subnet: $OUTPUT_DIR/viper-private-subnet.hcl"
log "  - Static IP:      $OUTPUT_DIR/viper-static-ip.hcl"