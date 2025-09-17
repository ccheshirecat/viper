# Hypervisor Setup Guide

This guide covers setting up the microVM infrastructure for Viper's browser automation framework.

## Architecture Overview

Viper uses **microVMs** for browser automation, providing:
- **Kernel-level isolation** for each browser session
- **Persistent state** with long-lived VMs containing browser sessions
- **Massive scalability** through lightweight Alpine-based VMs
- **Plugin system** for specialized workflows (casino, social media, etc.)

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Viper CLI     │───▶│  Nomad Cluster   │───▶│  libvirt + QEMU     │
│                 │    │                  │    │                     │
│ Plugins System  │    │ Job Orchestrator │    │ ┌─────────────────┐ │
│ • Casino        │    │ • VM Lifecycle   │    │ │   Alpine VM     │ │
│ • Social Media  │    │ • Health Checks  │    │ │ ┌─────────────┐ │ │
│ • E-commerce    │    │ • Load Balancing │    │ │ │ viper-agent │ │ │
│ • Custom...     │    │ • Auto-scaling   │    │ │ │  + Browser  │ │ │
└─────────────────┘    └──────────────────┘    │ │ └─────────────┘ │ │
                                               │ └─────────────────┘ │
                                               └─────────────────────┘
```

## Development Setup (macOS)

### 1. Install Dependencies

```bash
# Install libvirt and QEMU
brew install libvirt qemu

# Start libvirt daemon
brew services start libvirt

# Verify libvirt is running
virsh version
```

### 2. Install Nomad libvirt Driver

```bash
# Download the driver plugin
mkdir -p /opt/nomad/plugins
cd /opt/nomad/plugins

# Get latest version for macOS
wget https://releases.hashicorp.com/nomad-driver-virt/0.7.0/nomad-driver-virt_0.7.0_darwin_amd64.zip
unzip nomad-driver-virt_0.7.0_darwin_amd64.zip
chmod +x nomad-driver-virt
```

### 3. Configure Nomad

Create `/etc/nomad.d/client.hcl`:

```hcl
# Nomad Client Configuration
datacenter = "dc1"
data_dir   = "/opt/nomad/data"
log_level  = "INFO"

bind_addr = "0.0.0.0"

client {
  enabled = true

  # Plugin directory
  plugin_dir = "/opt/nomad/plugins"
}

# Enable libvirt driver plugin
plugin "nomad-driver-virt" {
  config {
    enabled  = true
    host_uri = "qemu:///system"  # System-level libvirt for shared VMs
  }
}

server {
  enabled          = true
  bootstrap_expect = 1
}
```

### 4. Setup VM Networking

```bash
# Create libvirt default network (if not exists)
virsh net-define /dev/stdin <<EOF
<network>
  <name>default</name>
  <bridge name="virbr0"/>
  <forward mode="nat"/>
  <ip address="192.168.122.1" netmask="255.255.255.0">
    <dhcp>
      <range start="192.168.122.2" end="192.168.122.254"/>
    </dhcp>
  </ip>
</network>
EOF

# Start the network
virsh net-start default
virsh net-autostart default
```

## Production Setup (Linux)

### 1. Cloud Hypervisor Integration

```bash
# Install Cloud Hypervisor
curl -L https://github.com/cloud-hypervisor/cloud-hypervisor/releases/latest/download/cloud-hypervisor-static -o /usr/local/bin/cloud-hypervisor
chmod +x /usr/local/bin/cloud-hypervisor

# Configure libvirt to use Cloud Hypervisor
# Note: libvirt 8.0+ has Cloud Hypervisor support
```

### 2. Performance Tuning

```hcl
# Production Nomad configuration
plugin "nomad-driver-virt" {
  config {
    enabled  = true
    host_uri = "ch:///system"  # Cloud Hypervisor URI

    # Performance optimizations
    default_vcpu_topology = "sockets=1,cores=2,threads=1"
    default_memory_backing = "hugepages"
  }
}
```

## Plugin System Architecture

### Plugin Interface

Plugins define specialized workflows for browser automation:

```go
type ViperPlugin interface {
    // Plugin metadata
    Name() string
    Version() string
    Description() string

    // Lifecycle management
    Initialize(config PluginConfig) error
    Shutdown() error

    // Workload management
    CreateWorkload(ctx context.Context, req WorkloadRequest) (*WorkloadResponse, error)
    ListWorkloads(ctx context.Context) ([]WorkloadStatus, error)
    GetWorkload(ctx context.Context, id string) (*WorkloadStatus, error)
    UpdateWorkload(ctx context.Context, id string, req WorkloadUpdateRequest) error
    DeleteWorkload(ctx context.Context, id string) error

    // Custom actions
    ExecuteAction(ctx context.Context, workloadID string, action string, params map[string]interface{}) (*ActionResponse, error)
}
```

### Example: Casino Plugin

```go
type CasinoPlugin struct {
    nomadClient *nomad.Client
    workloads   map[string]*CasinoWorkload
    config      CasinoPluginConfig
}

type CasinoWorkload struct {
    ID            string
    VMName        string
    AgentURL      string
    BrowserCtx    string
    Profile       CasinoProfile
    Status        string
    LastActivity  time.Time
}

func (p *CasinoPlugin) ExecuteAction(ctx context.Context, workloadID string, action string, params map[string]interface{}) (*ActionResponse, error) {
    workload := p.workloads[workloadID]

    switch action {
    case "claim_bonus":
        return p.claimBonus(ctx, workload)
    case "check_balance":
        return p.checkBalance(ctx, workload)
    case "place_bet":
        amount := params["amount"].(float64)
        return p.placeBet(ctx, workload, amount)
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
}

func (p *CasinoPlugin) claimBonus(ctx context.Context, w *CasinoWorkload) (*ActionResponse, error) {
    // Execute JavaScript in the VM's browser context
    agentReq := &agent.ExecuteJSRequest{
        ContextID: w.BrowserCtx,
        Script: `
            // Navigate to bonus page
            await page.goto('/promotions/daily-bonus');

            // Click claim button
            await page.click('[data-testid="claim-bonus-btn"]');

            // Wait for confirmation
            await page.waitForSelector('[data-testid="bonus-claimed"]');

            // Return result
            return {
                success: true,
                amount: await page.textContent('[data-testid="bonus-amount"]')
            };
        `,
    }

    resp, err := p.callAgent(ctx, w.AgentURL, agentReq)
    if err != nil {
        return nil, err
    }

    return &ActionResponse{
        Success: true,
        Data:    resp.Result,
    }, nil
}
```

### Plugin CLI Integration

```bash
# Install and enable a plugin
viper plugins install casino-plugin

# Create a workload pool
viper workloads create casino-pool \
    --plugin casino \
    --profile stake-profile.json \
    --count 5 \
    --memory 2048 \
    --cpus 2

# Execute plugin actions
viper workloads action casino-pool claim_bonus
viper workloads action casino-pool check_balance
viper workloads action casino-pool place_bet --amount 10.50

# Monitor workloads
viper workloads list
viper workloads status casino-pool
```

## VM Image Management

Our Packer-built Alpine images contain:
- **Minimal Alpine Linux** (~50MB)
- **Chromium browser** with automation capabilities
- **viper-agent** HTTP API server
- **Automatic service startup** via OpenRC

VM lifecycle:
1. **Boot** → Alpine starts, runs viper-agent service
2. **Ready** → Agent exposes HTTP API on port 8080
3. **Work** → Browser contexts created, profiles loaded
4. **Persist** → Long-lived VMs maintain session state
5. **Scale** → Nomad manages VM pool automatically

This architecture enables unprecedented scalability and isolation for browser automation workloads.