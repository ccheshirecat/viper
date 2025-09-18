# Viper Troubleshooting Guide

**Comprehensive troubleshooting guide for Viper microVM browser automation issues.**

---

## Quick Diagnostic Commands

```bash
# System health check
viper debug system

# Check Nomad cluster status
nomad node status
nomad job status

# Verify libvirt functionality
virsh list --all
virsh capabilities

# Check VM agent connectivity
curl -s http://VM_IP:8080/health | jq
```

---

## Installation Issues

### 1. Build Failures

**Issue**: `make build` fails with Go compilation errors

**Symptoms**:
```
go: module requires Go 1.21 or later
build failed: exit status 1
```

**Solution**:
```bash
# Check Go version
go version

# Install/upgrade Go if needed
# macOS
brew install go

# Ubuntu/Debian
sudo apt update && sudo apt install golang-1.21

# Update PATH if necessary
export PATH=/usr/lib/go-1.21/bin:$PATH
```

**Issue**: `make rootfs-build` fails with Packer errors

**Symptoms**:
```
Error: invalid accelerator kvm
Build 'alpine' errored: Build was halted
```

**Solutions**:

**On macOS**:
```bash
# Fix accelerator in alpine.pkr.hcl
accelerator = "tcg"    # Use software emulation

# Or install QEMU with HVF support
brew uninstall qemu
brew install --HEAD qemu  # Development version with HVF
```

**On Linux**:
```bash
# Verify KVM support
kvm-ok
ls -la /dev/kvm

# Add user to KVM group
sudo usermod -aG kvm $USER
sudo usermod -aG libvirt $USER
newgrp kvm
```

### 2. Dependency Issues

**Issue**: Nomad virt driver not found

**Symptoms**:
```
[ERROR] client.driver_mgr: failed to fingerprint driver: driver=virt error="plugin not found"
```

**Solution**:
```bash
# Linux: Install official driver
sudo apt install nomad-driver-virt

# macOS: Use QEMU exec driver (development)
# Use jobs/dev-vm-macos.nomad.hcl template instead

# Verify plugin installation
nomad node status -verbose | grep virt
```

**Issue**: libvirt permission denied

**Symptoms**:
```
libvirt: QEMU Driver error : internal error: permission denied
```

**Solution**:
```bash
# Fix permissions
sudo usermod -aG libvirt $USER
sudo usermod -aG qemu $USER

# Restart libvirt
sudo systemctl restart libvirtd

# Check socket permissions
ls -la /var/run/libvirt/libvirt-sock
```

---

## VM Creation Issues

### 1. VM Fails to Start

**Issue**: VM creation hangs or fails immediately

**Diagnostic Commands**:
```bash
# Check Nomad job status
nomad job status viper-vm-<name>
nomad alloc status <alloc-id>
nomad alloc logs <alloc-id>

# Check VM in libvirt
virsh list --all
virsh dominfo <vm-name>
```

**Common Causes & Solutions**:

**Insufficient Resources**:
```bash
# Check host resources
free -h
df -h
lscpu

# Reduce VM resource allocation
viper vms create test --memory 1024 --cpus 1
```

**Image Path Issues**:
```bash
# Verify rootfs image exists
ls -la /var/lib/virt/images/viper-rootfs-latest.qcov2

# Check image integrity
qemu-img info /var/lib/virt/images/viper-rootfs-latest.qcov2
qemu-img check /var/lib/virt/images/viper-rootfs-latest.qcov2

# Fix permissions
sudo chown libvirt-qemu:libvirt-qemu /var/lib/virt/images/*
```

**Network Configuration**:
```bash
# Check libvirt networks
virsh net-list --all

# Start default network if needed
sudo virsh net-start default
sudo virsh net-autostart default

# Create custom network if needed
sudo virsh net-define viper-network.xml
sudo virsh net-start viper-network
```

### 2. VM Boots but Agent Unreachable

**Issue**: VM starts but viper-agent doesn't respond

**Diagnostic Commands**:
```bash
# Check VM console (if accessible)
virsh console <vm-name>

# Check Nomad service health
nomad job status viper-vm-<name>
nomad alloc status <alloc-id> | grep Health

# Network connectivity test
nmap -p 8080 <vm-ip>
telnet <vm-ip> 8080
```

**Common Causes & Solutions**:

**Agent Service Not Started**:
```bash
# If you can access VM console:
# Check if agent binary exists
ls -la /usr/local/bin/viper-agent

# Check service status (inside VM)
rc-service viper-agent status
rc-service viper-agent start

# Check agent logs (inside VM)
tail -f /var/log/viper/agent.log
```

**Firewall Issues**:
```bash
# Check host firewall
sudo ufw status
sudo iptables -L | grep 8080

# Allow VM traffic
sudo ufw allow from 192.168.122.0/24 to any port 8080

# Check VM firewall (inside VM if accessible)
iptables -L
```

**Port Forwarding Problems** (macOS/QEMU):
```bash
# Check if port is bound
netstat -tlnp | grep 8080
lsof -i :8080

# Verify QEMU args include port forwarding
ps aux | grep qemu | grep hostfwd
```

---

## Browser Automation Issues

### 1. Browser Context Creation Fails

**Issue**: `viper browsers spawn` returns errors

**Diagnostic Commands**:
```bash
# Test agent connectivity
curl -s http://<vm-ip>:8080/health

# Try spawning context with verbose output
curl -v -X POST http://<vm-ip>:8080/spawn/test-ctx

# Check agent logs
viper debug agent <vm-name> --logs
```

**Common Causes & Solutions**:

**Chromium Not Available**:
```bash
# Check if Chromium installed in rootfs
# Rebuild rootfs with explicit Chromium installation

# Verify in VM console (if accessible)
which chromium-browser
chromium-browser --version
```

**Display Issues**:
```bash
# Ensure headless mode
# Agent should start Chromium with --headless flag

# Check X11/Wayland requirements
# Alpine VM should not require display server
```

**Resource Constraints**:
```bash
# Increase VM memory
viper vms destroy <vm-name>
viper vms create <vm-name> --memory 2048 --cpus 2

# Check VM resource usage
virsh domstats <vm-name> --cpu-total --memory
```

### 2. Task Execution Fails

**Issue**: Tasks submit but fail to complete

**Diagnostic Commands**:
```bash
# Get task logs
viper tasks logs <vm-name> <task-id>

# Check task screenshots
viper tasks screenshots <vm-name> <task-id>

# Verify task file format
cat task.json | jq
```

**Common Solutions**:

**Invalid Task Format**:
```json
{
  "id": "test-task-001",
  "vm_id": "test-vm",
  "url": "https://example.com",
  "timeout": 60000
}
```

**Network Connectivity**:
```bash
# Test external connectivity from VM
# If you can access VM console:
ping 8.8.8.8
wget -O- https://example.com

# Check DNS resolution
nslookup example.com
```

**Timeout Issues**:
```json
{
  "timeout": 120000  // Increase timeout to 2 minutes
}
```

---

## Performance Issues

### 1. Slow VM Boot Times

**Issue**: VMs take too long to start

**Diagnostic Data**:
```bash
# Time VM creation
time viper vms create speed-test

# Check host I/O performance
iostat -x 1 5
iotop -a

# Monitor VM startup
virsh domstats <vm-name> --cpu-total --block
```

**Optimizations**:

**Storage Performance**:
```bash
# Use SSD for VM images
sudo mkdir -p /ssd/virt/images
sudo mv /var/lib/virt/images/* /ssd/virt/images/
sudo ln -sf /ssd/virt/images /var/lib/virt/images

# Optimize qcow2 format
qemu-img create -f qcow2 -o cluster_size=2M,lazy_refcounts=on optimized.qcov2 10G
```

**CPU Configuration**:
```bash
# Enable CPU host-passthrough (Linux only)
# Modify job template to use host CPU features

# Check CPU features
lscpu | grep Flags
cat /proc/cpuinfo | grep flags
```

**Memory Configuration**:
```bash
# Enable hugepages
echo 1024 | sudo tee /proc/sys/vm/nr_hugepages

# Configure libvirt for hugepages
# Add to /etc/libvirt/qemu.conf:
# hugetlbfs_mount = "/dev/hugepages"
```

### 2. Browser Automation Slow

**Issue**: Browser tasks take too long to execute

**Diagnostic Commands**:
```bash
# Check VM resource usage during task
virsh domstats <vm-name> --cpu-total --memory --block

# Monitor network usage
virsh domstats <vm-name> --net

# Time individual operations
time curl -X POST http://<vm-ip>:8080/task -d @task.json
```

**Optimizations**:

**Increase VM Resources**:
```bash
# More CPU and memory for complex pages
viper vms create high-perf --memory 4096 --cpus 4
```

**Browser Flags**:
```go
// In agent implementation, add performance flags
chromedp.Flag("disable-backgrounding-occluded-windows", true),
chromedp.Flag("disable-background-timer-throttling", true),
chromedp.Flag("disable-renderer-backgrounding", true),
chromedp.Flag("disable-features", "TranslateUI,VizDisplayCompositor"),
```

---

## Networking Issues

### 1. VM Cannot Access Internet

**Issue**: Browser tasks fail to load external URLs

**Diagnostic Commands**:
```bash
# Test VM network connectivity
# If accessible via console:
ping 8.8.8.8
wget -O- https://httpbin.org/ip

# Check host NAT/forwarding
sudo iptables -t nat -L
cat /proc/sys/net/ipv4/ip_forward
```

**Solutions**:

**Enable IP Forwarding**:
```bash
# Temporary
sudo sysctl net.ipv4.ip_forward=1

# Permanent
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

**Fix NAT Rules**:
```bash
# Check existing rules
sudo iptables -t nat -L POSTROUTING

# Add NAT rule if missing
sudo iptables -t nat -A POSTROUTING -s 192.168.122.0/24 -o eth0 -j MASQUERADE

# Make persistent (Ubuntu)
sudo apt install iptables-persistent
sudo netfilter-persistent save
```

**DNS Resolution**:
```bash
# Check VM DNS configuration
# Inside VM (if accessible):
cat /etc/resolv.conf

# Fix DNS in libvirt network
virsh net-edit default
# Add: <dns><forwarder addr="8.8.8.8"/></dns>
```

### 2. Cannot Connect to VM Agent

**Issue**: CLI commands fail with connection refused

**Diagnostic Commands**:
```bash
# Check VM IP address
virsh domifaddr <vm-name>

# Test port accessibility
nmap -p 8080 <vm-ip>
nc -v <vm-ip> 8080

# Check Nomad port mapping
nomad alloc status <alloc-id> | grep Port
```

**Solutions**:

**Service Discovery Issues**:
```bash
# Get VM IP from Nomad
nomad alloc status <alloc-id> -json | jq '.Resources.Networks[0].IP'

# Update CLI to use correct IP
viper debug network --vm <vm-name>
```

**Port Configuration**:
```bash
# Verify agent listens on correct port
# Inside VM (if accessible):
netstat -tlnp | grep 8080
ss -tlnp | grep 8080
```

---

## Data and Storage Issues

### 1. Task Results Missing

**Issue**: Cannot retrieve screenshots or logs

**Diagnostic Commands**:
```bash
# Check task storage directory
# Inside VM (if accessible):
ls -la /var/viper/tasks/

# Check disk space
df -h
du -sh /var/viper/
```

**Solutions**:

**Storage Full**:
```bash
# Increase VM disk size
qemu-img resize vm-disk.qcov2 +2G

# Clean old task data
# Inside VM:
find /var/viper/tasks -mtime +7 -exec rm -rf {} \;
```

**Permission Issues**:
```bash
# Fix task directory permissions
# Inside VM:
sudo chown -R viper:viper /var/viper/
sudo chmod -R 755 /var/viper/
```

### 2. Session Persistence Problems

**Issue**: Browser state not maintained between tasks

**Diagnostic Commands**:
```bash
# Check browser profile directory
# Inside VM:
ls -la /home/viper/.config/chromium/

# Verify VM isn't being recreated
nomad job history viper-vm-<name>
```

**Solutions**:

**VM Recreation**:
```bash
# Ensure VM uses restart policy, not recreation
# Check job template for proper restart_policy

# Use long-lived VMs
nomad job stop viper-vm-<name>
# Update job with longer max_client_disconnect
```

**Profile Storage**:
```bash
# Verify profile injection worked
curl -s http://<vm-ip>:8080/profile/ctx-1 | jq

# Re-inject profile if needed
viper profiles attach <vm> ctx-1 profile.json
```

---

## Cluster and Scaling Issues

### 1. VM Scheduling Failures

**Issue**: VMs not scheduled on cluster nodes

**Diagnostic Commands**:
```bash
# Check node resources
nomad node status -verbose

# Check job constraints
nomad job inspect viper-vm-<name>

# Check scheduling reasons
nomad eval status <eval-id>
```

**Solutions**:

**Resource Exhaustion**:
```bash
# Free up resources
nomad job stop old-job

# Add more nodes
nomad node status | wc -l

# Adjust resource requirements
# Reduce memory/CPU in job templates
```

**Node Constraints**:
```bash
# Check node attributes
nomad node status -verbose <node-id>

# Update job constraints
# Remove overly restrictive constraints
# Add node_class = "viper-worker" if needed
```

### 2. High Availability Issues

**Issue**: Single point of failure

**Solutions**:

**Multi-Server Setup**:
```bash
# Deploy 3+ Nomad servers
# Configure proper bootstrap_expect
# Use consul for discovery
```

**Load Balancing**:
```bash
# HAProxy for Nomad API
# Multiple client nodes
# Distribute VM workloads
```

---

## Monitoring and Debugging

### Advanced Debugging

**Enable Debug Logging**:
```bash
# Nomad debug logs
export NOMAD_LOG_LEVEL=DEBUG
sudo systemctl restart nomad

# Agent debug logs
# Modify agent to use --log-level debug
```

**Performance Profiling**:
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Add profiling to agent:
import _ "net/http/pprof"
```

**Network Debugging**:
```bash
# Packet capture
sudo tcpdump -i virbr0 port 8080

# Network tracing
strace -e trace=network curl http://vm-ip:8080/health
```

---

## Getting Help

### Log Collection

**Create Support Bundle**:
```bash
#!/bin/bash
# Collect diagnostic information

SUPPORT_DIR="viper-support-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$SUPPORT_DIR"

# System information
uname -a > "$SUPPORT_DIR/system-info.txt"
lscpu > "$SUPPORT_DIR/cpu-info.txt"
free -h > "$SUPPORT_DIR/memory-info.txt"
df -h > "$SUPPORT_DIR/disk-info.txt"

# Nomad information
nomad node status > "$SUPPORT_DIR/nomad-nodes.txt"
nomad job status > "$SUPPORT_DIR/nomad-jobs.txt"

# Libvirt information
virsh list --all > "$SUPPORT_DIR/virt-domains.txt"
virsh net-list --all > "$SUPPORT_DIR/virt-networks.txt"

# Logs
journalctl -u nomad --since "1 hour ago" > "$SUPPORT_DIR/nomad.log"
journalctl -u libvirtd --since "1 hour ago" > "$SUPPORT_DIR/libvirt.log"

# Configuration
cp /etc/nomad.d/* "$SUPPORT_DIR/" 2>/dev/null || true

tar czf "$SUPPORT_DIR.tar.gz" "$SUPPORT_DIR"
echo "Support bundle created: $SUPPORT_DIR.tar.gz"
```

### Community Resources

- **GitHub Issues**: [Report bugs and feature requests](https://github.com/ccheshirecat/viper/issues)
- **Discussions**: [Community support and questions](https://github.com/ccheshirecat/viper/discussions)
- **Documentation**: [Additional guides and examples](https://github.com/ccheshirecat/viper/docs)

---

**This troubleshooting guide covers the most common issues encountered when deploying and operating Viper in production environments.**