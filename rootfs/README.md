# Viper Rootfs - Alpine Linux VM Images

This directory contains the Packer template and build system for creating minimal, production-ready Alpine Linux images with the embedded Viper agent for microVM environments.

## Prerequisites

- **Packer**: Install from [packer.io](https://www.packer.io/downloads)
- **QEMU**: Required for building VM images
- **viper-agent binary**: Must be built first (`make build-agent`)

## Quick Start

```bash
# Validate the Packer template
make rootfs-validate

# Build standard rootfs image
make rootfs-build

# Build GPU-enabled rootfs image
make rootfs-build-gpu

# Prepare release artifacts
make rootfs-release

# Show information about built images
make rootfs-info

# Clean build artifacts
make rootfs-clean
```

## Template Features

### alpine.pkr.hcl

Production-ready Packer HCL template with the following capabilities:

- **Base System**: Alpine Linux 3.19 minimal installation
- **Browser Engine**: Chromium with full automation support
- **Agent Integration**: viper-agent binary embedded and configured as system service
- **Security Hardening**: Minimal attack surface, unnecessary services disabled
- **GPU Support**: Optional GPU drivers and acceleration libraries
- **Service Management**: OpenRC configuration for automatic agent startup
- **Optimization**: Compressed qcow2 format, minimal disk usage

### Build Configuration

The template supports several build variables:

- `version`: Version tag for the image (default: "latest")
- `disk_size`: Disk size in MB (default: 2048)
- `memory`: Build-time memory allocation (default: 1024)
- `enable_gpu`: Enable GPU support (default: false)
- `alpine_version`: Alpine Linux version (default: "3.19")

### Output Format

Built images are in qcow2 format, compatible with:
- **QEMU/KVM**: Direct usage
- **Cloud Hypervisor**: Native support
- **Firecracker**: With format conversion
- **Nomad**: Via libvirt driver

## Directory Structure

After build completion:

```
dist/rootfs/
├── viper-rootfs-{version}-{timestamp}/
│   ├── viper-rootfs-{version}-{timestamp}.qcow2  # Main VM image
│   └── metadata.json                              # Build metadata
└── release/                                       # Release artifacts
    ├── viper-rootfs-*.qcow2                      # Images
    ├── viper-rootfs-*.qcow2.sha256               # Checksums
    └── metadata.json                              # Metadata
```

## Usage Examples

### Development

```bash
# Quick validation during development
make rootfs-validate

# Build development image
make rootfs-build
```

### Production

```bash
# Build production release with GPU support
make rootfs-build-gpu

# Prepare complete release package
make rootfs-release
```

### VM Deployment

```bash
# QEMU/KVM
qemu-system-x86_64 \
  -m 2048 \
  -hda dist/rootfs/release/viper-rootfs-latest.qcow2 \
  -netdev user,id=net0,hostfwd=tcp::8080-:8080 \
  -device virtio-net,netdev=net0

# Agent will be accessible at localhost:8080
```

## Image Components

### Installed Packages

- **Base**: bash, curl, wget, ca-certificates, tzdata
- **Browser**: chromium, chromium-chromedriver
- **Graphics**: mesa-dri-gallium, mesa-gl, fonts
- **GPU** (optional): mesa-vulkan-*, linux-firmware-*
- **System**: dbus, supervisor

### Directory Layout

- `/usr/local/bin/viper-agent`: Agent binary (executable)
- `/var/viper/tasks/`: Task storage directory (viper:viper ownership)
- `/var/log/viper/`: Log directory
- `/etc/init.d/viper-agent`: OpenRC service script

### Network Configuration

- Agent listens on port 8080
- Automatic startup via OpenRC
- Network isolation through VM boundaries

## Customization

The Packer template can be customized by:

1. **Modifying variables** in the template
2. **Adding provisioners** for additional software
3. **Extending build configuration** in the Makefile

## Security Features

- Minimal package installation
- Disabled unnecessary services
- User/group isolation (viper:viper)
- Network isolation via VM boundaries
- Log rotation and management
- Service hardening via OpenRC

## Troubleshooting

### Build Issues

```bash
# Check Packer installation
packer version

# Validate template syntax
packer validate alpine.pkr.hcl

# Check agent binary exists
ls -la ../bin/viper-agent
```

### Image Issues

```bash
# Check built images
make rootfs-info

# Test image boot (requires QEMU)
qemu-system-x86_64 -m 1024 -hda path/to/image.qcow2
```

---

*Built with the Viper Engineering Doctrine: Production-ready from day one*