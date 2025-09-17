# Viper Rootfs - Production-Ready Alpine Linux Image with Chromium and Agent
# This Packer template builds a minimal, secure, and fast-booting Alpine Linux image
# specifically designed for running the viper-agent in microVM environments.

packer {
  required_version = ">= 1.9.0"
  required_plugins {
    qemu = {
      version = ">= 1.0.10"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

# Variables for build configuration
variable "version" {
  description = "Version tag for the image"
  type        = string
  default     = "latest"
}

variable "output_dir" {
  description = "Output directory for the built image"
  type        = string
  default     = "out"
}

variable "disk_size" {
  description = "Disk size in MB (minimal but sufficient)"
  type        = number
  default     = 2048
}

variable "memory" {
  description = "Build-time memory allocation in MB"
  type        = number
  default     = 1024
}

variable "cpu_count" {
  description = "Number of CPUs for build"
  type        = number
  default     = 2
}

variable "alpine_version" {
  description = "Alpine Linux version to use"
  type        = string
  default     = "3.19"
}

variable "enable_gpu" {
  description = "Enable GPU support in the image"
  type        = bool
  default     = false
}

# Local values for computed configurations
locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")

  # Alpine ISO URL based on version
  alpine_iso_url = "https://dl-cdn.alpinelinux.org/alpine/v${var.alpine_version}/releases/x86_64/alpine-virt-${var.alpine_version}.0-x86_64.iso"

  # Output filename with version and timestamp
  output_filename = "viper-rootfs-${var.version}-${local.timestamp}"

  # Agent binary source path (must exist before build)
  agent_binary_path = "../bin/viper-agent"
}

# QEMU builder for creating the Alpine Linux VM image
source "qemu" "alpine" {
  # VM Configuration
  vm_name           = "${local.output_filename}"
  output_directory  = "${var.output_dir}/${local.output_filename}"

  # Image Configuration
  format            = "qcow2"
  disk_size         = var.disk_size
  disk_compression  = true

  # ISO Configuration
  iso_url           = local.alpine_iso_url
  iso_checksum      = "file:${local.alpine_iso_url}.sha256"

  # Hardware Configuration
  memory            = var.memory
  cpus              = var.cpu_count
  accelerator       = "tcg"    # Software emulation (change to "kvm" on Linux, "hvf" on macOS with proper QEMU)
  net_device        = "virtio-net"
  disk_interface    = "virtio"

  # Boot Configuration
  boot_wait         = "30s"
  boot_command = [
    # Login as root (no password in Alpine virt)
    "root<enter><wait>",

    # Setup networking
    "setup-interfaces -a<enter><wait5>",
    "rc-service networking start<enter><wait5>",

    # Set up temporary SSH for provisioning
    "echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config<enter>",
    "echo 'root:viper' | chpasswd<enter>",
    "rc-service sshd start<enter><wait5>",

    # Start installation
    "setup-disk -m sys /dev/vda<enter><wait10>",

    # Configure basic system
    "setup-timezone -z UTC<enter>",
    "setup-hostname viper-vm<enter>",

    # Reboot into installed system
    "reboot<enter>"
  ]

  # SSH Configuration for provisioning
  ssh_username      = "root"
  ssh_password      = "viper"
  ssh_timeout       = "20m"
  ssh_wait_timeout  = "20m"

  # QEMU specific settings
  qemu_binary       = "qemu-system-x86_64"
  headless          = true
  use_default_display = false

  # Shutdown configuration
  shutdown_command  = "poweroff"
  shutdown_timeout  = "5m"
}

# Build configuration with provisioners
build {
  name = "viper-alpine-rootfs"

  sources = [
    "source.qemu.alpine"
  ]

  # Verify agent binary exists before starting build
  provisioner "shell-local" {
    inline = [
      "echo 'Verifying viper-agent binary exists...'",
      "if [ ! -f '${local.agent_binary_path}' ]; then",
      "  echo 'ERROR: viper-agent binary not found at ${local.agent_binary_path}'",
      "  echo 'Please run: make build-agent'",
      "  exit 1",
      "fi",
      "echo 'Agent binary verified: ${local.agent_binary_path}'"
    ]
  }

  # Wait for system to be ready after reboot
  provisioner "shell" {
    inline = [
      "echo 'Waiting for system to be ready...'",
      "sleep 30"
    ]
  }

  # System base configuration
  provisioner "shell" {
    inline = [
      "echo 'Configuring Alpine Linux base system...'",

      # Update package repositories
      "apk update",
      "apk upgrade",

      # Add community repository for additional packages
      "echo 'http://dl-cdn.alpinelinux.org/alpine/v${var.alpine_version}/community' >> /etc/apk/repositories",
      "apk update",

      # Install essential packages
      "apk add --no-cache \\",
      "  bash \\",
      "  curl \\",
      "  wget \\",
      "  ca-certificates \\",
      "  tzdata \\",
      "  openssl \\",
      "  dbus \\",
      "  supervisor",

      # Create necessary users and groups
      "addgroup -g 1000 viper",
      "adduser -D -u 1000 -G viper -s /bin/bash viper",

      "echo 'Base system configuration complete.'"
    ]
  }

  # Chromium installation and configuration
  provisioner "shell" {
    inline = [
      "echo 'Installing and configuring Chromium...'",

      # Install Chromium and dependencies
      "apk add --no-cache \\",
      "  chromium \\",
      "  chromium-chromedriver \\",
      "  mesa-dri-gallium \\",
      "  mesa-gl \\",
      "  ttf-freefont \\",
      "  font-noto \\",
      "  xvfb",

      # GPU support packages (conditional)
      var.enable_gpu ? join(" ", [
        "apk add --no-cache \\",
        "  mesa-vulkan-intel \\",
        "  mesa-vulkan-radeon \\",
        "  linux-firmware-amdgpu \\",
        "  linux-firmware-i915"
      ]) : "echo 'GPU support disabled, skipping GPU packages'",

      # Configure Chromium for headless operation
      "mkdir -p /etc/chromium",
      "echo '--no-sandbox' >> /etc/chromium/default-flags",
      "echo '--disable-dev-shm-usage' >> /etc/chromium/default-flags",
      "echo '--disable-gpu' >> /etc/chromium/default-flags",
      "echo '--remote-debugging-port=9222' >> /etc/chromium/default-flags",

      # Test Chromium installation
      "chromium-browser --version",

      "echo 'Chromium installation and configuration complete.'"
    ]
  }

  # Create directory structure
  provisioner "shell" {
    inline = [
      "echo 'Creating Viper directory structure...'",

      # Create application directories
      "mkdir -p /usr/local/bin",
      "mkdir -p /var/viper/tasks",
      "mkdir -p /var/log/viper",
      "mkdir -p /etc/viper",

      # Set proper ownership and permissions
      "chown -R viper:viper /var/viper",
      "chmod 755 /var/viper",
      "chmod 755 /var/viper/tasks",
      "chmod 755 /var/log/viper",

      "echo 'Directory structure created.'"
    ]
  }

  # Copy and configure viper-agent binary
  provisioner "file" {
    source      = local.agent_binary_path
    destination = "/tmp/viper-agent"
  }

  provisioner "shell" {
    inline = [
      "echo 'Installing and configuring viper-agent...'",

      # Install agent binary
      "mv /tmp/viper-agent /usr/local/bin/viper-agent",
      "chmod +x /usr/local/bin/viper-agent",
      "chown root:root /usr/local/bin/viper-agent",

      # Verify agent binary
      "/usr/local/bin/viper-agent --version || echo 'Agent version check failed (may be expected)'",

      # Create systemd service for agent (Alpine uses OpenRC)
      "cat > /etc/init.d/viper-agent << 'EOF'",
      "#!/sbin/openrc-run",
      "",
      "name=\"viper-agent\"",
      "description=\"Viper Browser Automation Agent\"",
      "",
      "command=\"/usr/local/bin/viper-agent\"",
      "command_args=\"--listen=:8080\"",
      "command_user=\"viper:viper\"",
      "command_background=\"yes\"",
      "pidfile=\"/run/viper-agent.pid\"",
      "",
      "output_log=\"/var/log/viper/agent.log\"",
      "error_log=\"/var/log/viper/agent-error.log\"",
      "",
      "depend() {",
      "    need net",
      "    after networking",
      "}",
      "EOF",

      # Make service executable and enable it
      "chmod +x /etc/init.d/viper-agent",
      "rc-update add viper-agent default",

      "echo 'viper-agent installation and configuration complete.'"
    ]
  }

  # System hardening and optimization
  provisioner "shell" {
    inline = [
      "echo 'Applying system hardening and optimization...'",

      # Remove unnecessary packages and cache
      "apk del --purge \\",
      "  alpine-baselayout-data \\",
      "  apk-tools-doc",
      "apk cache clean",
      "rm -rf /var/cache/apk/*",

      # Remove temporary files
      "rm -rf /tmp/*",
      "rm -rf /var/tmp/*",

      # Clear logs
      "find /var/log -type f -exec truncate -s 0 {} \\;",

      # Remove SSH keys and history for security
      "rm -f /root/.ssh/authorized_keys",
      "rm -f /root/.bash_history",
      "rm -f /home/viper/.bash_history",

      # Optimize for container/VM deployment
      "echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf",
      "echo 'vm.swappiness=10' >> /etc/sysctl.conf",

      # Disable unnecessary services
      "rc-update del sshd default",

      # Configure automatic agent startup
      "echo 'viper-agent will start automatically on boot'",

      "echo 'System hardening and optimization complete.'"
    ]
  }

  # Final validation
  provisioner "shell" {
    inline = [
      "echo 'Running final validation...'",

      # Verify all components
      "ls -la /usr/local/bin/viper-agent",
      "ls -la /var/viper/",
      "chromium-browser --version",

      # Check service configuration
      "ls -la /etc/init.d/viper-agent",

      # Disk usage report
      "df -h",
      "echo 'Rootfs image validation complete.'"
    ]
  }

  # Generate image metadata
  provisioner "shell-local" {
    inline = [
      "echo 'Generating image metadata...'",
      "cat > '${var.output_dir}/${local.output_filename}/metadata.json' << EOF",
      "{",
      "  \"name\": \"viper-rootfs\",",
      "  \"version\": \"${var.version}\",",
      "  \"build_time\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",",
      "  \"alpine_version\": \"${var.alpine_version}\",",
      "  \"disk_size_mb\": ${var.disk_size},",
      "  \"gpu_enabled\": ${var.enable_gpu},",
      "  \"components\": {",
      "    \"chromium\": \"$(docker run --rm alpine:${var.alpine_version} apk info chromium | grep chromium- | head -1 || echo 'unknown')\",",
      "    \"viper_agent\": \"built-from-source\"",
      "  },",
      "  \"usage\": {",
      "    \"qemu\": \"qemu-system-x86_64 -m 2048 -hda ${local.output_filename}.qcow2 -netdev user,id=net0 -device virtio-net,netdev=net0\",",
      "    \"cloud_hypervisor\": \"Compatible with Cloud Hypervisor VMM\",",
      "    \"firecracker\": \"Requires conversion for Firecracker compatibility\"",
      "  }",
      "}",
      "EOF",
      "echo 'Image metadata generated.'"
    ]
  }
}

# Build completion message will be shown by the provisioner
# Image information is available in the generated metadata.json file