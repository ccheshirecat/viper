# Viper Rootfs - Production-Ready Alpine Linux via libvirt
# Uses libvirt abstraction layer for hypervisor independence

packer {
  required_version = ">= 1.9.0"
  required_plugins {
    libvirt = {
      version = ">= 0.5.0"
      source  = "github.com/thomasklein94/libvirt"
    }
  }
}

# Variables
variable "version" {
  description = "Version tag for the image"
  type        = string
  default     = "latest"
}

variable "memory" {
  description = "Memory in MB"
  type        = number
  default     = 2048
}

variable "cpus" {
  description = "Number of CPUs"
  type        = number
  default     = 2
}

variable "disk_size" {
  description = "Disk size in MB"
  type        = number
  default     = 4096
}

# Local values
locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
  vm_name = "viper-alpine-${var.version}-${local.timestamp}"
  alpine_iso = "https://dl-cdn.alpinelinux.org/alpine/v3.22/releases/aarch64/alpine-standard-3.22.1-aarch64.iso"
  agent_binary_path = "../bin/viper-agent"
  output_dir = "out"
}

# libvirt builder
source "libvirt" "alpine" {
  libvirt_uri = "qemu:///system"

  # VM Configuration
  vm_name = local.vm_name
  memory = var.memory
  vcpus = var.cpus

  # Disk Configuration
  volume_name = "${local.vm_name}.qcow2"
  volume_size = "${var.disk_size}M"
  volume_format = "qcow2"

  # Boot from ISO
  boot_devices = ["cdrom", "hd"]

  # Network
  network_interface {
    type = "network"
    network = "default"
  }

  # Graphics (for VNC access during build)
  graphics {
    type = "vnc"
    listen_address = "127.0.0.1"
    port = 5900
  }

  # Boot commands for Alpine installation
  boot_wait = "30s"
  boot_command = [
    # Login as root
    "root<enter><wait5>",

    # Set up networking
    "setup-interfaces -a<enter><wait10>",
    "rc-service networking start<enter><wait5>",

    # Install to disk
    "setup-disk -m sys /dev/vda<enter><wait30>",

    # Set password for SSH access
    "echo 'root:viper' | chpasswd<enter>",
    "rc-service sshd start<enter><wait5>",

    # Reboot to installed system
    "reboot<enter>"
  ]

  # SSH configuration
  ssh_username = "root"
  ssh_password = "viper"
  ssh_timeout = "20m"

  # Output
  output_directory = local.output_dir
  shutdown_command = "poweroff"
}

# Build configuration
build {
  name = "viper-alpine-libvirt"

  sources = ["source.libvirt.alpine"]

  # Wait for system to be ready
  provisioner "shell" {
    inline = [
      "echo 'System ready, starting provisioning...'",
      "sleep 10"
    ]
  }

  # Install base packages
  provisioner "shell" {
    inline = [
      "apk update && apk upgrade",
      "apk add --no-cache bash curl wget ca-certificates",
      "apk add --no-cache chromium xvfb dbus supervisor",
      "mkdir -p /var/viper/tasks /var/log/viper /etc/viper",
      "adduser -D -s /bin/bash viper"
    ]
  }

  # Copy viper-agent
  provisioner "file" {
    source = local.agent_binary_path
    destination = "/tmp/viper-agent"
  }

  # Install and configure agent
  provisioner "shell" {
    inline = [
      "mv /tmp/viper-agent /usr/local/bin/viper-agent",
      "chmod +x /usr/local/bin/viper-agent",
      "chown root:root /usr/local/bin/viper-agent",

      # Create OpenRC service
      "cat > /etc/init.d/viper-agent << 'EOF'",
      "#!/sbin/openrc-run",
      "name=\"viper-agent\"",
      "command=\"/usr/local/bin/viper-agent\"",
      "command_args=\"--listen=:8080\"",
      "command_user=\"viper:viper\"",
      "command_background=\"yes\"",
      "pidfile=\"/run/viper-agent.pid\"",
      "depend() { need net; }",
      "EOF",

      "chmod +x /etc/init.d/viper-agent",
      "rc-update add viper-agent default"
    ]
  }

  # Cleanup
  provisioner "shell" {
    inline = [
      "apk cache clean",
      "rm -rf /tmp/* /var/tmp/*",
      "echo 'libvirt-based Alpine rootfs build complete'"
    ]
  }
}