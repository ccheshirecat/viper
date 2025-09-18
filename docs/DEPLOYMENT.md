# Viper Deployment Guide

**Production-grade deployment configurations for Viper microVM browser automation.**

---

## Overview

This guide covers deploying Viper in production environments with proper scaling, monitoring, and high availability. Viper's architecture supports both single-node development setups and multi-node production clusters.

## Prerequisites

### Hardware Requirements

**Minimum Production Node:**
- **CPU**: 16 cores with VT-x/AMD-V support
- **Memory**: 32GB RAM (16GB for VMs + 16GB for host)
- **Storage**: 500GB SSD for VM images and task data
- **Network**: 1Gbps network interface

**Recommended Production Node:**
- **CPU**: 32+ cores with VT-x/AMD-V support
- **Memory**: 64GB+ RAM
- **Storage**: 1TB+ NVMe SSD
- **Network**: 10Gbps network interface
- **GPU**: Optional for hardware-accelerated rendering

### Software Requirements

**Operating System:**
- Ubuntu 20.04 LTS or newer
- CentOS 8 or RHEL 8+
- Debian 11+ (Bullseye)

**Core Dependencies:**
- Linux kernel 5.0+ with KVM support
- libvirt 6.0+
- QEMU 5.0+
- Docker 20.10+ (for containerized services)

---

## Single Node Setup

### 1. System Preparation

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install KVM and virtualization tools
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils

# Verify KVM support
kvm-ok

# Add user to libvirt groups
sudo usermod -aG libvirt,kvm $USER
```

### 2. Install HashiCorp Tools

```bash
# Add HashiCorp GPG key and repository
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list

# Install Nomad and Packer
sudo apt update
sudo apt install -y nomad packer

# Install nomad-driver-virt
sudo apt install -y nomad-driver-virt
```

### 3. Configure Nomad

Create `/etc/nomad.d/nomad.hcl`:

```hcl
datacenter = "dc1"
data_dir   = "/opt/nomad/data"
log_level  = "INFO"

bind_addr = "0.0.0.0"

server {
  enabled = true
  bootstrap_expect = 1
}

client {
  enabled = true
  servers = ["127.0.0.1:4647"]

  # Configure resource allocation
  reserved {
    cpu    = 2000  # Reserve 2 cores for host
    memory = 4096  # Reserve 4GB for host
  }
}

# Enable virt driver plugin
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

# UI configuration
ui_config {
  enabled = true
}
```

### 4. Start Services

```bash
# Create required directories
sudo mkdir -p /opt/nomad/data /var/lib/virt/images

# Start and enable services
sudo systemctl start libvirtd nomad
sudo systemctl enable libvirtd nomad

# Verify services
sudo systemctl status libvirtd nomad
nomad node status
```

### 5. Deploy Viper

```bash
# Clone and build Viper
git clone https://github.com/ccheshirecat/viper.git
cd viper

# Build binaries and rootfs
make build
make rootfs-build

# Copy rootfs to image directory
sudo cp out/viper-rootfs-*.qcow2 /var/lib/virt/images/
sudo ln -sf /var/lib/virt/images/viper-rootfs-*.qcow2 /var/lib/virt/images/viper-rootfs-latest.qcow2

# Test VM creation
./bin/viper vms create test-vm --memory 2048 --cpus 2
```

---

## Multi-Node Cluster Setup

### 1. Cluster Architecture

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Nomad Server 1 │  │  Nomad Server 2 │  │  Nomad Server 3 │
│  (Leader)       │  │  (Follower)     │  │  (Follower)     │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
    ┌─────────────────────────┼─────────────────────────┐
    │                         │                         │
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ Nomad Client 1  │  │ Nomad Client 2  │  │ Nomad Client N  │
│ (Worker Node)   │  │ (Worker Node)   │  │ (Worker Node)   │
│ ├── libvirt     │  │ ├── libvirt     │  │ ├── libvirt     │
│ ├── VMs (1-20)  │  │ ├── VMs (1-20)  │  │ ├── VMs (1-20)  │
│ └── virt driver │  │ └── virt driver │  │ └── virt driver │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 2. Server Node Configuration

**Nomad Server** (`/etc/nomad.d/server.hcl`):

```hcl
datacenter = "dc1"
data_dir   = "/opt/nomad/data"
log_level  = "INFO"

bind_addr = "{{ GetPrivateInterfaces | include \"network\" \"10.0.0.0/8\" | attr \"address\" }}"

server {
  enabled = true
  bootstrap_expect = 3  # Number of server nodes

  # Cluster join
  retry_join = ["10.0.1.10", "10.0.1.11", "10.0.1.12"]

  # Encryption
  encrypt = "base64-encoded-key"
}

# Disable client on server nodes
client {
  enabled = false
}

ui_config {
  enabled = true
}
```

### 3. Client Node Configuration

**Nomad Client** (`/etc/nomad.d/client.hcl`):

```hcl
datacenter = "dc1"
data_dir   = "/opt/nomad/data"
log_level  = "INFO"

bind_addr = "{{ GetPrivateInterfaces | include \"network\" \"10.0.0.0/8\" | attr \"address\" }}"

# Disable server on client nodes
server {
  enabled = false
}

client {
  enabled = true
  servers = ["10.0.1.10:4647", "10.0.1.11:4647", "10.0.1.12:4647"]

  # Node class for workload targeting
  node_class = "viper-worker"

  # Resource configuration
  reserved {
    cpu    = 4000   # Reserve 4 cores for host
    memory = 8192   # Reserve 8GB for host
    disk   = 10240  # Reserve 10GB for host
  }
}

plugin "nomad-driver-virt" {
  config {
    enabled    = true
    data_dir   = "/var/lib/virt"
    image_paths = ["/var/lib/virt/images", "/shared/images"]

    emulator {
      uri = "qemu:///system"
    }

    # Resource limits per VM
    default_resources {
      cpu_limit    = 2000  # 2 cores max per VM
      memory_limit = 4096  # 4GB max per VM
    }
  }
}
```

### 4. Load Balancer Configuration

**HAProxy** (`/etc/haproxy/haproxy.cfg`):

```
global
    daemon
    maxconn 4096

defaults
    mode http
    timeout connect 5000ms
    timeout client  50000ms
    timeout server  50000ms

# Nomad UI Load Balancer
frontend nomad_ui
    bind *:80
    default_backend nomad_servers

backend nomad_servers
    balance roundrobin
    server nomad1 10.0.1.10:4646 check
    server nomad2 10.0.1.11:4646 check
    server nomad3 10.0.1.12:4646 check

# Nomad API Load Balancer
frontend nomad_api
    bind *:4647
    default_backend nomad_api_servers

backend nomad_api_servers
    balance roundrobin
    server nomad1 10.0.1.10:4647 check
    server nomad2 10.0.1.11:4647 check
    server nomad3 10.0.1.12:4647 check
```

---

## Container Deployment

### 1. Docker Compose Setup

**Development Stack** (`docker-compose.yml`):

```yaml
version: '3.8'

services:
  nomad:
    image: hashicorp/nomad:1.6
    command: nomad agent -dev -bind 0.0.0.0 -log-level INFO
    ports:
      - "4646:4646"  # UI
      - "4647:4647"  # API
      - "4648:4648"  # RPC
    volumes:
      - nomad_data:/nomad/data
      - /var/run/docker.sock:/var/run/docker.sock
      - ./jobs:/jobs
    environment:
      - NOMAD_ADDR=http://0.0.0.0:4647
    privileged: true

  viper-cli:
    build:
      context: .
      dockerfile: docker/Dockerfile.cli
    volumes:
      - ./examples:/examples
      - ./profiles:/profiles
    depends_on:
      - nomad
    environment:
      - NOMAD_ADDR=http://nomad:4647

volumes:
  nomad_data:
```

### 2. Production Kubernetes Deployment

**Nomad Server StatefulSet** (`k8s/nomad-server.yaml`):

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nomad-server
spec:
  serviceName: nomad-server
  replicas: 3
  selector:
    matchLabels:
      app: nomad-server
  template:
    metadata:
      labels:
        app: nomad-server
    spec:
      containers:
      - name: nomad
        image: hashicorp/nomad:1.6
        args:
          - nomad
          - agent
          - -config=/etc/nomad.d/
        ports:
        - containerPort: 4646
        - containerPort: 4647
        - containerPort: 4648
        volumeMounts:
        - name: config
          mountPath: /etc/nomad.d
        - name: data
          mountPath: /nomad/data
        env:
        - name: NOMAD_LOCAL_CONFIG
          value: |
            datacenter = "k8s"
            data_dir = "/nomad/data"
            server {
              enabled = true
              bootstrap_expect = 3
            }
      volumes:
      - name: config
        configMap:
          name: nomad-config
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi
```

---

## Monitoring and Observability

### 1. Prometheus Configuration

**Nomad Metrics** (`prometheus.yml`):

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  # Nomad server metrics
  - job_name: 'nomad-server'
    static_configs:
      - targets: ['nomad-server:4646']
    metrics_path: /v1/metrics
    params:
      format: ['prometheus']

  # Nomad client metrics
  - job_name: 'nomad-client'
    static_configs:
      - targets: ['nomad-client-1:4646', 'nomad-client-2:4646']
    metrics_path: /v1/metrics
    params:
      format: ['prometheus']

  # Viper agent metrics (from VMs)
  - job_name: 'viper-agents'
    consul_sd_configs:
      - server: 'consul:8500'
        services: ['viper-agent']
```

### 2. Grafana Dashboards

**VM Resource Usage Dashboard**:
- CPU utilization per VM
- Memory usage and allocation
- Disk I/O and storage usage
- Network traffic patterns
- Task success/failure rates

**Nomad Cluster Dashboard**:
- Cluster node status
- Job scheduling metrics
- Resource allocation
- Health check status

### 3. Log Aggregation

**Fluentd Configuration** (`fluent.conf`):

```
<source>
  @type tail
  path /var/log/nomad/*.log
  pos_file /var/log/fluentd/nomad.log.pos
  tag nomad.*
  format json
</source>

<source>
  @type tail
  path /var/viper/tasks/*/stdout.log
  pos_file /var/log/fluentd/viper-tasks.log.pos
  tag viper.tasks
  format none
</source>

<match nomad.**>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name nomad-logs
</match>

<match viper.**>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name viper-logs
</match>
```

---

## Security Hardening

### 1. Network Security

**Firewall Rules** (UFW):

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow Nomad cluster communication
sudo ufw allow from 10.0.0.0/8 to any port 4647,4648

# Allow Nomad UI (restrict to admin subnet)
sudo ufw allow from 10.0.100.0/24 to any port 4646

# Allow VM traffic (libvirt bridge)
sudo ufw allow in on virbr0
sudo ufw allow out on virbr0

# Enable firewall
sudo ufw enable
```

**Network Policies**:

```bash
# Isolate VM networks
sudo virsh net-define - <<EOF
<network>
  <name>viper-isolated</name>
  <bridge name='viperbr0' stp='on' delay='0'/>
  <ip address='192.168.100.1' netmask='255.255.255.0'>
    <dhcp>
      <range start='192.168.100.100' end='192.168.100.200'/>
    </dhcp>
  </ip>
</network>
EOF

sudo virsh net-start viper-isolated
sudo virsh net-autostart viper-isolated
```

### 2. Access Control

**ACL Configuration** (`/etc/nomad.d/acl.hcl`):

```hcl
acl = {
  enabled = true
  token_ttl = "30m"
  policy_ttl = "60s"
}
```

**Policy Examples**:

```hcl
# Operator policy (full access)
namespace "*" {
  policy = "write"
}

node {
  policy = "write"
}

# Developer policy (limited access)
namespace "development" {
  policy = "write"
}

namespace "production" {
  policy = "read"
}
```

### 3. Secrets Management

**Vault Integration**:

```hcl
vault {
  enabled = true
  address = "https://vault.example.com:8200"
  token   = "vault-nomad-token"

  create_from_role = "nomad-cluster"
}
```

---

## Backup and Disaster Recovery

### 1. State Backup

```bash
#!/bin/bash
# Nomad state backup script

BACKUP_DIR="/backups/nomad/$(date +%Y-%m-%d-%H-%M-%S)"
mkdir -p "$BACKUP_DIR"

# Backup Nomad state
nomad operator snapshot save "$BACKUP_DIR/nomad-state.snap"

# Backup VM images
rsync -av /var/lib/virt/images/ "$BACKUP_DIR/vm-images/"

# Backup job definitions
cp -r /jobs/ "$BACKUP_DIR/jobs/"

# Create manifest
cat > "$BACKUP_DIR/manifest.txt" <<EOF
Backup created: $(date)
Nomad version: $(nomad version)
Node count: $(nomad node status | wc -l)
Jobs count: $(nomad job status | wc -l)
EOF
```

### 2. Recovery Procedures

**Restore from Backup**:

```bash
#!/bin/bash
# Disaster recovery script

BACKUP_DIR="/backups/nomad/2024-01-01-12-00-00"

# Stop Nomad
sudo systemctl stop nomad

# Restore state
nomad operator snapshot restore "$BACKUP_DIR/nomad-state.snap"

# Restore VM images
rsync -av "$BACKUP_DIR/vm-images/" /var/lib/virt/images/

# Start Nomad
sudo systemctl start nomad

# Verify cluster
nomad node status
nomad job status
```

---

## Performance Tuning

### 1. Host Optimization

**Kernel Parameters** (`/etc/sysctl.d/99-viper.conf`):

```
# VM memory overcommit
vm.overcommit_memory = 1
vm.overcommit_ratio = 100

# Network optimization
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000

# File descriptor limits
fs.file-max = 1000000
fs.nr_open = 1000000
```

**System Limits** (`/etc/security/limits.conf`):

```
* soft nofile 65535
* hard nofile 65535
* soft nproc 65535
* hard nproc 65535
```

### 2. VM Optimization

**libvirt Configuration** (`/etc/libvirt/qemu.conf`):

```
# CPU affinity for better performance
# vcpu_pin_set = "2-31"

# Memory backing
# hugetlbfs_mount = "/dev/hugepages"

# Security driver
security_driver = "selinux"

# User/group for QEMU processes
user = "libvirt-qemu"
group = "libvirt-qemu"
```

### 3. Storage Optimization

**VM Image Optimization**:

```bash
# Create optimized VM images
qemu-img create -f qcow2 -o cluster_size=2M,lazy_refcounts=on viper-optimized.qcow2 10G

# Preallocation for better performance
qemu-img create -f qcow2 -o preallocation=metadata viper-preallocated.qcow2 10G

# Compression for storage savings
qemu-img convert -c -f qcow2 -O qcow2 input.qcow2 compressed.qcow2
```

---

## Troubleshooting

### Common Issues

**VM Creation Fails**:
```bash
# Check libvirt status
sudo systemctl status libvirtd

# Check VM resources
virsh capabilities
virsh nodeinfo

# Check image permissions
ls -la /var/lib/virt/images/
sudo chown libvirt-qemu:libvirt-qemu /var/lib/virt/images/*
```

**Nomad Driver Issues**:
```bash
# Check driver status
nomad node status -verbose

# Check plugin logs
journalctl -u nomad -f | grep virt

# Restart nomad service
sudo systemctl restart nomad
```

**Performance Problems**:
```bash
# Check resource usage
htop
iotop
iftop

# Check VM performance
virsh domstats --cpu-total --balloon --block --perf
```

---

**This deployment guide ensures Viper runs reliably in production with proper monitoring, security, and scalability.**