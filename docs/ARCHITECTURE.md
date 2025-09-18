# Viper Architecture

**A comprehensive guide to Viper's microVM-based browser automation architecture.**

---

## Overview

Viper's architecture is built around the principle of **true isolation** through microVMs. Unlike container-based solutions that share the host kernel, Viper runs each browser session in a complete, isolated Alpine Linux virtual machine.

## Core Components

### 1. Viper CLI
**Location**: `cmd/viper/`
**Purpose**: Command-line interface for managing VMs, tasks, and plugins

The CLI serves as the primary interface for:
- **VM Lifecycle**: Creating, listing, and destroying microVMs
- **Task Management**: Submitting automation tasks and retrieving results
- **Plugin System**: Installing and managing specialized automation workflows
- **Debugging**: System diagnostics and troubleshooting

```go
// Core CLI architecture
type ViperCLI struct {
    nomadClient *nomad.Client      // Direct Nomad API integration
    templates   *TemplateParser    // HCL job template parser
    plugins     *PluginManager     // Plugin system interface
}
```

### 2. Viper Agent
**Location**: `cmd/agent/`
**Purpose**: HTTP API server running inside each microVM

The agent provides browser automation capabilities:
- **Browser Context Management**: Spawn/manage chromedp contexts
- **Task Execution**: Run automation scripts with screenshots/logs
- **Profile Injection**: Load cookies, localStorage, fingerprints
- **Health Monitoring**: Status endpoints for Nomad health checks

```go
// Agent architecture inside each VM
type Agent struct {
    server   *gin.Engine           // HTTP API server
    contexts map[string]*Context   // Browser contexts
    taskDir  string               // Task storage directory
}
```

### 3. Nomad Integration
**Location**: `internal/nomad/`
**Purpose**: Orchestration and VM lifecycle management

Nomad handles:
- **Job Scheduling**: Place VMs across cluster nodes
- **Resource Management**: CPU, memory, disk allocation
- **Health Checks**: Monitor VM and agent health
- **Service Discovery**: Route traffic to VM agents

```hcl
# Production job using virt driver
job "viper-vm" {
  task "microvm" {
    driver = "virt"  # Creates actual VMs

    config {
      image = "file:///path/to/viper-rootfs.qcow2"
      memory = "2048"
      vcpu = 2
    }
  }
}
```

### 4. Plugin System
**Location**: `internal/plugins/` (planned)
**Purpose**: Extensible automation workflows

Plugins provide domain-specific automation:
- **Casino Plugins**: Bonus claiming, betting strategies
- **Social Plugins**: Account management, content automation
- **E-commerce Plugins**: Price monitoring, inventory tracking
- **Custom Plugins**: User-defined automation workflows

## Architecture Diagrams

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           User Interface                             │
├─────────────────────────────────────────────────────────────────────┤
│  Viper CLI                                                          │
│  ├── VM Management (create, list, destroy)                         │
│  ├── Task Submission (submit, logs, screenshots)                   │
│  ├── Plugin System (install, workloads, actions)                   │
│  └── Debug Tools (system, network, agent)                          │
└─────────────────┬───────────────────────────────────────────────────┘
                  │ Nomad API + HTTP Requests
                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Orchestration Layer                           │
├─────────────────────────────────────────────────────────────────────┤
│  Nomad Cluster                                                     │
│  ├── Job Scheduler (place VMs across nodes)                       │
│  ├── Resource Manager (CPU, memory, network)                      │
│  ├── Health Monitor (agent checks, auto-restart)                  │
│  └── Service Discovery (route traffic to VMs)                     │
└─────────────────┬───────────────────────────────────────────────────┘
                  │ libvirt API (virt driver)
                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Virtualization Layer                          │
├─────────────────────────────────────────────────────────────────────┤
│  libvirt + Hypervisor                                             │
│  ├── VM Creation (QEMU/KVM, Cloud Hypervisor)                     │
│  ├── Network Management (bridges, NAT, port forwarding)           │
│  ├── Storage Management (qcow2 images, snapshots)                 │
│  └── Hardware Passthrough (GPU, USB devices)                      │
└─────────────────┬───────────────────────────────────────────────────┘
                  │ Boot VMs with Alpine rootfs
                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      microVM Instances                             │
├─────────────────────────────────────────────────────────────────────┤
│  Alpine Linux VM 1          Alpine Linux VM 2      Alpine VM N    │
│  ├── viper-agent:8080       ├── viper-agent:8081   ├── agent:808N │
│  ├── Chromium browser       ├── Chromium browser   ├── Chromium   │
│  ├── Browser contexts       ├── Browser contexts   ├── Contexts   │
│  └── Task storage           └── Task storage       └── Storage    │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow Architecture

```
┌──────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   User   │───▶│ Viper CLI   │───▶│   Nomad     │───▶│   libvirt   │
└──────────┘    └─────────────┘    └─────────────┘    └─────────────┘
     │               │                    │                  │
     │               │                    │                  ▼
     │               │                    │            ┌──────────┐
     │               │                    │            │   VM 1   │
     │               │                    │            │  Agent   │
     │               │                    │            └──────────┘
     │               │                    │                  │
     │               │                    │                  ▼
     │               │                    │            ┌──────────┐
     │               │                    │            │ Chromium │
     │               │                    │            │ Browser  │
     │               │                    │            └──────────┘
     │               │                    │                  │
     │               ▼                    ▼                  ▼
     └─────────── HTTP Requests ──────────────── Browser Automation
                (screenshots, logs)
```

## Isolation Model

### Kernel-Level Isolation

Each browser session runs in a complete microVM with:

```
┌─────────────────────────────────────────────────────────────┐
│                      Host System                           │
├─────────────────────────────────────────────────────────────┤
│  Linux Kernel (Host)                                       │
│  ├── KVM Hypervisor                                       │
│  ├── libvirt Management                                   │
│  └── Nomad Orchestration                                  │
└─────────────────┬───────────────────────────────────────────┘
                  │ Hardware Virtualization
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    VM 1 (Isolated)                         │
├─────────────────────────────────────────────────────────────┤
│  Alpine Linux Kernel (Guest)                               │
│  ├── Separate Memory Space (2GB)                          │
│  ├── Independent Filesystem (qcow2)                       │
│  ├── Isolated Network Stack (virtio-net)                  │
│  ├── Dedicated CPU Cores (2 vCPUs)                       │
│  └── viper-agent + Chromium                               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    VM N (Isolated)                         │
├─────────────────────────────────────────────────────────────┤
│  Alpine Linux Kernel (Guest)                               │
│  ├── Separate Memory Space                                 │
│  ├── Independent Filesystem                                │
│  ├── Isolated Network Stack                               │
│  └── Completely Independent Browser Session               │
└─────────────────────────────────────────────────────────────┘
```

**Benefits of microVM isolation:**
- **No Shared Kernel**: Each VM has its own Linux kernel
- **Memory Protection**: Hardware-enforced memory boundaries
- **Filesystem Isolation**: Separate rootfs prevents cross-contamination
- **Network Isolation**: VM-level networking with NAT/bridge
- **Process Namespace**: Complete separation of process trees

### Session Persistence

VM state persistence across operations:

```
┌────────────────────────────────────────────────────────────┐
│                   Persistent State                         │
├────────────────────────────────────────────────────────────┤
│  Browser Session Data:                                    │
│  ├── Cookies (authentication, preferences)                │
│  ├── localStorage (application state)                     │
│  ├── sessionStorage (temporary data)                      │
│  ├── IndexedDB (client-side databases)                    │
│  └── Cache (images, scripts, data)                        │
│                                                            │
│  VM System State:                                         │
│  ├── Browser profile (/home/viper/.config)                │
│  ├── Download directory (/home/viper/Downloads)           │
│  ├── Extension data (/home/viper/.config/extensions)      │
│  └── System logs (/var/log/viper/)                        │
│                                                            │
│  Task Results:                                             │
│  ├── Screenshots (/var/viper/tasks/*/screenshots/)        │
│  ├── Execution logs (/var/viper/tasks/*/stdout.log)       │
│  └── Metadata (/var/viper/tasks/*/metadata.json)          │
└────────────────────────────────────────────────────────────┘
```

## Network Architecture

### Development Environment (macOS)

```
┌──────────────┐    ┌─────────────────┐    ┌───────────────┐
│ Viper CLI    │───▶│ Nomad Agent     │───▶│ QEMU Process  │
│ (localhost)  │    │ (localhost)     │    │ (user mode)   │
└──────────────┘    └─────────────────┘    └───────────────┘
                                                   │
                                                   ▼
                                         ┌───────────────┐
                                         │ Alpine VM     │
                                         │ Agent :8080   │
                                         │ (port fwd)    │
                                         └───────────────┘
                                                   │
                                            HTTP requests
                                                   ▼
                                         ┌───────────────┐
                                         │ Chromium      │
                                         │ Browser       │
                                         └───────────────┘
```

### Production Environment (Linux)

```
┌──────────────┐    ┌─────────────────┐    ┌───────────────┐
│ Viper CLI    │───▶│ Nomad Cluster   │───▶│ libvirt Host  │
│ (remote)     │    │ (multi-node)    │    │ (KVM enabled) │
└──────────────┘    └─────────────────┘    └───────────────┘
                                                   │
                                                   ▼
                                         ┌───────────────┐
                                         │ Bridge Network│
                                         │ (virbr0)      │
                                         └───────────────┘
                                                   │
                                                   ▼
                                         ┌───────────────┐
                                         │ Alpine VM     │
                                         │ Agent :8080   │
                                         │ (bridged net) │
                                         └───────────────┘
```

## Storage Architecture

### Rootfs Image Management

```
┌─────────────────────────────────────────────────────────────┐
│                    Image Build Pipeline                     │
├─────────────────────────────────────────────────────────────┤
│  1. Packer Template (alpine.pkr.hcl)                       │
│     ├── Alpine Linux 3.19 base                            │
│     ├── Chromium browser installation                      │
│     ├── viper-agent binary embedding                       │
│     └── System hardening and optimization                  │
│                                                            │
│  2. Build Process (make rootfs-build)                      │
│     ├── Download Alpine ISO                                │
│     ├── Boot VM and provision                              │
│     ├── Install packages and configure                     │
│     └── Generate qcow2 image                               │
│                                                            │
│  3. Distribution (make rootfs-release)                     │
│     ├── Generate checksums                                 │
│     ├── Create release artifacts                           │
│     └── Deploy to image registry                           │
└─────────────────────────────────────────────────────────────┘
```

### VM Storage Layout

```
/var/lib/virt/images/
├── viper-rootfs-v0.1.0.qcow2        # Base VM image (read-only)
├── viper-rootfs-v0.1.0.qcow2.sha256 # Integrity verification
└── viper-rootfs-latest.qcow2        # Symlink to current version

Per-VM Instance Storage:
/var/lib/virt/instances/<vm-id>/
├── disk.qcow2                       # VM disk (copy-on-write)
├── metadata.json                    # VM configuration
└── snapshots/                       # VM state snapshots
    ├── pre-task.qcow2
    └── checkpoint.qcov2
```

## Security Model

### Attack Surface Reduction

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Boundaries                      │
├─────────────────────────────────────────────────────────────┤
│  Hardware Level:                                           │
│  ├── CPU virtualization extensions (VT-x/AMD-V)           │
│  ├── Memory management unit (MMU) isolation               │
│  └── IOMMU for device passthrough                         │
│                                                            │
│  Hypervisor Level:                                         │
│  ├── KVM kernel module (hardware-assisted)                │
│  ├── QEMU process isolation (separate user)               │
│  └── SELinux/AppArmor mandatory access control            │
│                                                            │
│  Guest OS Level:                                           │
│  ├── Alpine Linux (minimal attack surface)                │
│  ├── No SSH server (agent-only access)                    │
│  ├── Read-only rootfs with overlay                        │
│  └── Restricted user permissions (viper user)             │
│                                                            │
│  Application Level:                                        │
│  ├── viper-agent (minimal HTTP API)                       │
│  ├── Chromium sandboxing (seccomp, namespaces)           │
│  └── Task-specific resource limits                        │
└─────────────────────────────────────────────────────────────┘
```

### Threat Model

**Threats Mitigated:**
- **Browser Exploitation**: Contained within VM boundary
- **Privilege Escalation**: Limited to VM scope
- **Data Exfiltration**: Network isolation prevents lateral movement
- **Persistence**: VMs can be destroyed and recreated from clean images

**Threats Accepted:**
- **Host Compromise**: Would affect all VMs (mitigated by cluster deployment)
- **Hypervisor Escape**: Rare but possible (mitigated by security updates)
- **Side-Channel Attacks**: Theoretical (mitigated by CPU features)

## Performance Characteristics

### Resource Requirements

```
Per microVM Resource Usage:
├── CPU: 2 vCPUs (2000 MHz allocation)
├── Memory: 2GB VM + 512MB host overhead = 2.5GB total
├── Storage: 1GB base image + 512MB overlay = 1.5GB total
└── Network: Virtualized NIC with bridge/NAT

Host Scaling Capacity:
├── 32 CPU cores → 16 concurrent VMs (2 vCPUs each)
├── 64GB RAM → 25 concurrent VMs (2.5GB each)
├── 500GB SSD → 300+ VMs (1.5GB each)
└── Network: 1Gbps shared across all VMs
```

### Performance Benchmarks

```
VM Boot Time:
├── Cold boot (image load): ~15-30 seconds
├── Agent startup: ~5-10 seconds
├── Browser ready: ~5 seconds
└── Total ready time: ~30-45 seconds

Browser Automation:
├── Page navigation: 1-5 seconds
├── Screenshot capture: 500ms-2s
├── DOM interaction: 100-500ms
└── Task completion: 10-120 seconds (task dependent)

Concurrent Operations:
├── 10 VMs: Linear scaling
├── 50 VMs: 95% efficiency
├── 100+ VMs: Diminishing returns (I/O bound)
```

## Deployment Patterns

### Single Node Development

```
┌─────────────────────────────────────────────────────────────┐
│                    Development Setup                        │
├─────────────────────────────────────────────────────────────┤
│  Host: macOS with QEMU                                     │
│  ├── Nomad: Single agent (dev mode)                       │
│  ├── Scheduler: Local only                                │
│  ├── VMs: QEMU processes (software emulation)             │
│  └── Networking: User-mode NAT                            │
│                                                            │
│  Limitations:                                              │
│  ├── Single point of failure                              │
│  ├── Limited by host resources                            │
│  ├── No high availability                                 │
│  └── Software emulation (slower)                          │
└─────────────────────────────────────────────────────────────┘
```

### Production Cluster

```
┌─────────────────────────────────────────────────────────────┐
│                   Production Cluster                        │
├─────────────────────────────────────────────────────────────┤
│  Nomad Server Nodes (3x):                                 │
│  ├── Raft consensus (leader election)                     │
│  ├── Job scheduling decisions                             │
│  ├── State management                                     │
│  └── API endpoints                                        │
│                                                            │
│  Nomad Client Nodes (N x):                                │
│  ├── VM execution (libvirt + KVM)                         │
│  ├── Resource reporting                                   │
│  ├── Health monitoring                                    │
│  └── Log/metric collection                                │
│                                                            │
│  Infrastructure:                                           │
│  ├── Load balancer (HAProxy/nginx)                        │
│  ├── Shared storage (NFS/Ceph)                            │
│  ├── Monitoring (Prometheus/Grafana)                      │
│  └── Logging (ELK/Loki)                                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Future Architecture Enhancements

### Planned Improvements

1. **GPU Acceleration**: Hardware-accelerated rendering via GPU passthrough
2. **Snapshot Management**: VM state checkpoints for rapid task restoration
3. **Image Optimization**: Multi-stage builds and layer caching for faster boots
4. **Network Policies**: Fine-grained network isolation and traffic shaping
5. **Auto-scaling**: Dynamic VM provisioning based on workload demand

### Plugin Architecture Evolution

```
┌─────────────────────────────────────────────────────────────┐
│                   Plugin Ecosystem                         │
├─────────────────────────────────────────────────────────────┤
│  Core Framework:                                           │
│  ├── Plugin SDK (Go interfaces)                           │
│  ├── Plugin registry and discovery                        │
│  ├── Lifecycle management (install/upgrade/remove)        │
│  └── Resource isolation and sandboxing                    │
│                                                            │
│  Plugin Types:                                             │
│  ├── Task Plugins (automation workflows)                  │
│  ├── Profile Plugins (session management)                 │
│  ├── Analytics Plugins (data collection)                  │
│  └── Integration Plugins (external APIs)                  │
└─────────────────────────────────────────────────────────────┘
```

---

**This architecture enables Viper to provide unparalleled browser automation capabilities through true microVM isolation, perfect session persistence, and unlimited scalability.**