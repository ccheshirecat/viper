# Viper

**A microVM-based browser automation framework for unparalleled session persistence, kernel-level security, and massive scalability.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/ccheshirecat/viper)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/github/license/ccheshirecat/viper)](LICENSE)
[![Build Status](https://img.shields.io/github/workflow/status/ccheshirecat/viper/CI)](https://github.com/ccheshirecat/viper/actions)

---

## Overview

Viper revolutionizes browser automation by running each browser session in its own isolated Alpine Linux microVM. This approach provides kernel-level security, perfect session persistence, and unlimited scalability for complex, stateful browser tasks where stealth, reliability, and performance are paramount.

### Key Features

- **🔒 Kernel-Level Isolation**: Each browser session runs in a separate microVM with complete OS isolation
- **📱 Persistent Sessions**: Long-lived VMs maintain cookies, localStorage, and session state indefinitely
- **⚡ Massive Scalability**: Nomad orchestration enables thousands of concurrent browser sessions
- **🔌 Plugin System**: Specialized workloads for gaming, social media, e-commerce, and custom automation
- **🎯 Production Ready**: Built-in monitoring, logging, screenshots, and debugging capabilities
- **🌐 Multi-Platform**: Supports QEMU (development), Cloud Hypervisor, and Firecracker (production)

### Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Viper CLI     │───▶│  Nomad Cluster   │───▶│   libvirt Host      │
│   + Plugins     │    │ + microVM Jobs   │    │                     │
└─────────────────┘    └──────────────────┘    │ ┌─────────────────┐ │
                                               │ │   Alpine VM 1   │ │
Plugin-Driven Workloads:                      │ │ ┌─────────────┐ │ │
• Casino automation                           │ │ │ viper-agent │ │ │
• Social media management                     │ │ │ + Chromium  │ │ │
• E-commerce monitoring                       │ │ │   :8080     │ │ │
• Custom browser tasks                        │ │ └─────────────┘ │ │
                                               │ └─────────────────┘ │
                                               │        ...          │
                                               │ ┌─────────────────┐ │
                                               │ │   Alpine VM N   │ │
                                               │ │  (1000s of VMs) │ │
                                               │ └─────────────────┘ │
                                               └─────────────────────┘
```

---

## Quick Start

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Nomad 1.6+** - [Install Nomad](https://www.nomadproject.io/downloads)
- **Packer 1.9+** - [Install Packer](https://www.packer.io/downloads)

**Linux (Production):**
- **libvirt + KVM** - Hardware acceleration for microVMs
- **nomad-driver-virt** - [Install virt driver](https://developer.hashicorp.com/nomad/plugins/drivers/virt/install)

**macOS (Development):**
- **QEMU + libvirt** - `brew install qemu libvirt`

### Installation

```bash
# Clone and build
git clone https://github.com/ccheshirecat/viper.git
cd viper
make build

# Build Alpine rootfs with embedded agent
make rootfs-build

# Start Nomad (development)
nomad agent -dev

# Create your first microVM
./bin/viper vms create demo --memory 2048 --cpus 2

# Spawn browser context and start automating
./bin/viper browsers spawn demo ctx-1
./bin/viper tasks submit demo examples/task.json
./bin/viper tasks screenshots demo task-123
```

### Basic Workflow

```bash
# 1. Create long-lived microVM with persistent browser
viper vms create my-session --memory 2048

# 2. Inject browser profile (cookies, sessions, fingerprints)
viper profiles attach my-session ctx-1 profiles/authenticated-user.json

# 3. Submit automation tasks
viper tasks submit my-session automation/scrape-data.json

# 4. Monitor and retrieve results
viper tasks logs my-session task-456
viper tasks screenshots my-session task-456

# 5. Session persists across reboots, maintains login state
```

---

## Plugin System

Viper's true power comes from specialized plugins that handle complex, domain-specific automation workflows.

### Plugin-Based Workloads

```bash
# Install casino automation plugin
viper plugins install github.com/viper-plugins/casino-stake

# Create persistent VM pool for stake.com
viper workloads create stake-farm \
  --plugin casino-stake \
  --count 10 \
  --memory 2048 \
  --profile profiles/stake-accounts.json

# Execute coordinated actions across all VMs
viper workloads action stake-farm claim_daily_bonus
viper workloads action stake-farm rain_collect --threshold 0.001
viper workloads action stake-farm bet_strategy --game dice --amount 0.01

# Monitor the entire farm
viper workloads status stake-farm
```

### Available Plugins

- **🎰 Casino Plugins**: Automated bonus claiming, rain collection, betting strategies
- **📱 Social Plugins**: Account management, content posting, engagement automation
- **🛒 E-commerce Plugins**: Price monitoring, inventory tracking, purchase automation
- **🔧 Custom Plugins**: Build domain-specific automation with the plugin SDK

---

## Architecture Deep Dive

### microVM Isolation

Unlike container-based solutions, Viper runs each browser session in a complete microVM:

- **Kernel Isolation**: Separate Linux kernel per browser session
- **Memory Isolation**: No shared memory between sessions
- **Filesystem Isolation**: Each VM has independent rootfs
- **Network Isolation**: VM-level networking with NAT/bridge
- **Process Isolation**: Complete process namespace separation

### Session Persistence

VMs maintain state across:
- **Browser Sessions**: Cookies, localStorage, sessionStorage
- **Login States**: Authenticated sessions persist indefinitely
- **Fingerprints**: Canvas, WebGL, audio context fingerprints
- **Extensions**: Browser extensions and their data
- **Downloads**: Files and browser cache

### Orchestration

Nomad handles VM lifecycle:
- **Scheduling**: Place VMs across cluster nodes
- **Health Checks**: Monitor VM and agent health
- **Auto-restart**: Handle VM failures gracefully
- **Resource Management**: CPU, memory, disk allocation
- **Service Discovery**: Route traffic to VM agents

---

## Command Reference

### VM Management

```bash
# Create microVM
viper vms create <name> [--memory MB] [--cpus N] [--gpu] [--vmm qemu|chv]

# List running VMs
viper vms list [--status running|stopped|all]

# Destroy VM
viper vms destroy <name> [--force]
```

### Browser Context Management

```bash
# Spawn browser context inside VM
viper browsers spawn <vm> <context-id>

# List contexts
viper browsers list <vm>

# Close context
viper browsers close <vm> <context-id>
```

### Task Automation

```bash
# Submit automation task
viper tasks submit <vm> <task.json>

# Monitor task progress
viper tasks status <vm> <task-id>

# Get task logs
viper tasks logs <vm> <task-id> [--follow]

# Download screenshots
viper tasks screenshots <vm> <task-id> [--download]
```

### Profile Management

```bash
# Attach browser profile to context
viper profiles attach <vm> <context-id> <profile.json>

# Export current profile
viper profiles export <vm> <context-id> <output.json>

# List available profiles
viper profiles list
```

### Plugin System

```bash
# Install plugin
viper plugins install <plugin-url>

# List installed plugins
viper plugins list

# Create workload pool
viper workloads create <pool> --plugin <name> [--count N]

# Execute plugin action
viper workloads action <pool> <action> [--params key=value]

# Monitor workload pool
viper workloads status <pool> [--detailed]
```

### Debugging & Monitoring

```bash
# System diagnostics
viper debug system

# Network connectivity
viper debug network [--vm <name>]

# Agent debugging
viper debug agent <vm> [--logs] [--metrics]
```

---

## Configuration

### Nomad Job Templates

Viper uses HCL templates for VM configuration:

**Production (Linux + virt driver):**
```hcl
job "viper-vm-production" {
  datacenters = ["dc1"]
  type        = "service"

  group "vm-group" {
    task "microvm" {
      driver = "virt"  # libvirt driver

      config {
        image  = "file:///path/to/viper-rootfs-latest.qcow2"
        memory = "2048"
        vcpu   = 2

        network_interface {
          type   = "bridge"
          source = "virbr0"
          model  = "virtio"
        }
      }

      resources {
        cpu    = 2000
        memory = 2560
      }
    }
  }
}
```

**Development (macOS + QEMU):**
```hcl
job "viper-vm-development" {
  group "vm-group" {
    task "qemu-vm" {
      driver = "exec"

      config {
        command = "/opt/homebrew/bin/qemu-system-x86_64"
        args = [
          "-m", "1024",
          "-drive", "file=${NOMAD_TASK_DIR}/viper-rootfs.qcow2,format=qcow2",
          "-netdev", "user,id=net0,hostfwd=tcp::${NOMAD_PORT_http}-:8080"
        ]
      }
    }
  }
}
```

### Agent Configuration

The viper-agent runs inside each VM:

```bash
# Default agent startup
/usr/local/bin/viper-agent \
  --listen=:8080 \
  --task-dir=/var/viper/tasks \
  --log-level=info
```

### Browser Profiles

Profiles inject session data into browser contexts:

```json
{
  "id": "authenticated-user",
  "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
  "cookies": [
    {
      "name": "session_token",
      "value": "abc123",
      "domain": ".example.com"
    }
  ],
  "localStorage": {
    "example.com": {
      "user_preferences": "{\"theme\":\"dark\"}"
    }
  }
}
```

---

## Production Deployment

### Nomad Cluster Setup

**1. Install nomad-driver-virt (Linux only):**
```bash
# Ubuntu/Debian
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) test" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install nomad-driver-virt
```

**2. Configure Nomad client:**
```hcl
# /etc/nomad.d/client.hcl
plugin "nomad-driver-virt" {
  config {
    enabled    = true
    data_dir   = "/var/lib/virt"
    image_paths = ["/var/lib/virt/images"]

    emulator {
      uri = "qemu:///system"
    }
  }
}
```

**3. Deploy rootfs images:**
```bash
# Build and distribute rootfs
make rootfs-build
make rootfs-release

# Copy to all Nomad nodes
rsync -av out/viper-rootfs-latest.qcow2 nomad-node:/var/lib/virt/images/
```

### Monitoring & Observability

Viper integrates with standard observability tools:

- **Metrics**: Prometheus metrics from agents
- **Logs**: Centralized logging via syslog/fluentd
- **Tracing**: OpenTelemetry support for request tracing
- **Health**: Nomad health checks + custom agent endpoints

---

## Development

### Building from Source

```bash
# Install dependencies
make deps

# Run tests
make test

# Build binaries
make build

# Build rootfs images
make rootfs-build

# Run full CI pipeline
make ci
```

### Project Structure

```
viper/
├── cmd/
│   ├── viper/          # CLI application
│   └── agent/          # VM agent
├── internal/
│   ├── cli/            # CLI command implementations
│   ├── agent/          # Agent HTTP server
│   ├── nomad/          # Nomad integration
│   └── types/          # Core types
├── rootfs/             # VM image building
│   ├── alpine.pkr.hcl  # Packer template
│   └── README.md       # Rootfs documentation
├── jobs/               # Nomad job templates
├── examples/           # Example tasks and profiles
├── docs/               # Additional documentation
└── tests/              # Test suites
```

### Testing

```bash
# Unit tests
make test-unit

# Integration tests (requires Nomad)
make test-integration

# End-to-end tests (requires VMs)
make test-e2e

# Benchmarks
make benchmark
```

---

## Troubleshooting

### Common Issues

**VM fails to start:**
```bash
# Check Nomad job status
nomad job status viper-vm-<name>

# Check VM logs
viper debug agent <vm-name> --logs

# Verify rootfs image
qemu-img info /path/to/rootfs.qcow2
```

**Agent not responding:**
```bash
# Test network connectivity
viper debug network --vm <name>

# Check agent health
curl http://VM_IP:8080/health

# Restart VM
viper vms destroy <name> && viper vms create <name>
```

**Browser automation fails:**
```bash
# Check browser context
viper browsers list <vm>

# View task logs
viper tasks logs <vm> <task-id> --follow

# Get screenshots for debugging
viper tasks screenshots <vm> <task-id> --download
```

### Performance Tuning

**VM Resource Allocation:**
- **CPU**: 2+ cores for responsive browser automation
- **Memory**: 2GB+ for complex pages with media
- **Disk**: 1GB+ for task storage and browser cache

**Host Requirements:**
- **KVM support**: `/dev/kvm` accessible for hardware acceleration
- **Memory**: 4GB+ host RAM per VM (including overhead)
- **Storage**: SSD recommended for VM images and task data

---

## Contributing

Viper follows the **[Engineering Doctrine](.claude/CLAUDE.md)** - every line of code must be production-ready from day one.

### Development Process

1. **Fork & Branch**: Create feature branch from `main`
2. **Implement**: Write production-grade code with tests
3. **Document**: Update relevant documentation
4. **Test**: Ensure all tests pass (`make ci`)
5. **Submit**: Open pull request with detailed description

### Code Standards

- **Go**: Follow standard Go conventions and `gofmt`
- **Tests**: Mandatory for all critical functionality
- **Documentation**: Code must be self-documenting
- **Security**: No secrets in code, security-first design

### Plugin Development

See **[Plugin SDK Documentation](docs/plugin-sdk.md)** for building custom plugins.

---

## License

Licensed under the **[MIT License](LICENSE)**.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/ccheshirecat/viper/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ccheshirecat/viper/discussions)
- **Documentation**: [docs/](docs/)

---

**Built with the Viper Engineering Doctrine: Production-ready from day one.**