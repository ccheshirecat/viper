packer {
  required_version = ">= 1.9.0"
  required_plugins {
    qemu = {
      version = ">= 1.0.10"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

source "qemu" "alpine-x86_64" {
  # x86_64 QEMU configuration
  qemu_binary       = "qemu-system-x86_64"
  machine_type      = "pc"
  accelerator       = "kvm"  # KVM for Linux production hosts

  # Alpine x86_64 ISO
  iso_url          = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/x86_64/alpine-virt-3.21.1-x86_64.iso"
  iso_checksum     = "sha256:4efcc0f56bc6c9fa5b5e6f0fa1e6b93b9dbd37b8a11afa3c9d59c4c6ebbc6dce"

  # VM specifications optimized for microVM
  disk_size        = "1024M"  # Minimal 1GB for microVM
  memory           = 512      # 512MB RAM for microVM
  format           = "qcow2"
  headless         = true

  # Network and disk configuration
  net_device       = "virtio-net"
  disk_interface   = "virtio"

  # SSH configuration for provisioning
  ssh_username     = "root"
  ssh_password     = "password"
  ssh_timeout      = "5m"

  # VM output settings
  vm_name          = "viper-vm-x86_64"
  vnc_bind_address = "127.0.0.1"  # Localhost only for security

  # Boot configuration with Alpine setup automation
  boot_wait        = "10s"
  boot_command     = [
    "<wait>root<enter>",
    "<wait2>setup-alpine<enter>",
    "alpine<enter>",           # hostname
    "eth0<enter>",            # network interface
    "dhcp<enter>",            # DHCP configuration
    "n<enter>",               # manual network config? no
    "<wait5>password<enter>password<enter>",  # root password
    "UTC<enter>",             # timezone
    "<enter>",                # proxy? none
    "<enter>",                # NTP? default
    "r<enter>",               # random mirror
    "<enter>",                # setup user? no
    "openssh<enter>",         # SSH server
    "yes<enter>",             # allow root SSH
    "none<enter>",            # disk? none for now
    "vda<enter>",             # which disk? vda
    "sys<enter>",             # how to use? sys
    "y<enter>",               # erase disk? yes
    "<wait1m><reboot><enter>" # reboot after install
  ]

  shutdown_command = "poweroff"
}

# Build configuration
build {
  name    = "viper-alpine-x86_64-rootfs"
  sources = ["source.qemu.alpine-x86_64"]

  # Verify viper-agent binary exists locally
  provisioner "shell-local" {
    inline = [
      "echo 'Verifying viper-agent binary exists...'",
      "if [ ! -f '../bin/viper-agent' ]; then",
      "  echo 'ERROR: viper-agent not found at ../bin/viper-agent'",
      "  echo 'Please build viper-agent first: cd .. && make build'",
      "  exit 1",
      "fi",
      "file ../bin/viper-agent | grep -q 'x86-64' || echo 'WARNING: viper-agent may not be x86_64 binary'"
    ]
  }

  # Update system and install essential packages
  provisioner "shell" {
    inline = [
      "echo 'Setting up Alpine Linux for microVM...'",
      "apk update",
      "apk add --no-cache bash curl wget ca-certificates tzdata",
      "apk add --no-cache chromium chromium-chromedriver",
      "echo 'Base system configured.'"
    ]
  }

  # Copy viper-agent binary into VM
  provisioner "file" {
    source      = "../bin/viper-agent"
    destination = "/tmp/viper-agent"
  }

  # Install viper-agent as PID 1 init system
  provisioner "shell" {
    inline = [
      "echo 'Installing viper-agent as init system...'",

      # Install agent binary
      "install -m 755 /tmp/viper-agent /usr/local/bin/viper-agent",
      "rm /tmp/viper-agent",

      # Create init symlink (viper-agent will be PID 1)
      "ln -sf /usr/local/bin/viper-agent /sbin/init",

      # Create minimal directory structure
      "mkdir -p /var/viper/tasks",
      "mkdir -p /var/log/viper",

      # Create basic environment for microVM
      "echo 'export PATH=/usr/local/bin:/usr/bin:/bin' >> /etc/profile",
      "echo 'export CHROME_BIN=/usr/bin/chromium-browser' >> /etc/profile",
      "echo 'export DISPLAY=:99' >> /etc/profile",

      # Enable necessary kernel modules for browser
      "echo 'kernel.unprivileged_userns_clone=1' >> /etc/sysctl.conf",

      "echo 'viper-agent installed as PID 1 init system.'"
    ]
  }

  # Final system optimization for microVM
  provisioner "shell" {
    inline = [
      "echo 'Optimizing for microVM deployment...'",

      # Clean package cache
      "apk cache clean",
      "rm -rf /var/cache/apk/*",

      # Remove unnecessary files to minimize image size
      "rm -rf /tmp/* /var/tmp/*",
      "rm -rf /usr/share/doc/* /usr/share/man/*",

      # Clear shell history
      "history -c",
      "rm -f /root/.ash_history",

      "echo 'microVM optimization complete.'"
    ]
  }

  # Create manifest file with build information
  provisioner "shell" {
    inline = [
      "echo 'Creating build manifest...'",
      "cat > /etc/viper-build-info <<EOF",
      "VIPER_BUILD_ARCH=x86_64",
      "VIPER_BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)",
      "VIPER_ALPINE_VERSION=3.21.1",
      "VIPER_INIT_SYSTEM=viper-agent",
      "EOF",
      "echo 'Build manifest created at /etc/viper-build-info'"
    ]
  }
}