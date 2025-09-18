# Viper Rootfs - Production-Ready Alpine Linux via libvirt
# Uses Alpine minirootfs as base and provisions it with viper-agent

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

# Local values
locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
  vm_name = "viper-alpine-${var.version}-${local.timestamp}"
  agent_binary_path = "../bin/viper-agent"
  output_dir = "out"
}

# libvirt builder using Alpine minirootfs
source "libvirt" "alpine" {
  libvirt_uri = "ch+ssh://user@your-linux-host/system"

  # VM Configuration
  vcpu = var.cpus
  memory = var.memory

  # Network interface
  network_interface {
    type = "managed"
    alias = "communicator"
  }

  # SSH communicator
  communicator {
    communicator = "ssh"
    ssh_username = "root"
    ssh_password = "viper"
    ssh_timeout = "10m"
  }
  network_address_source = "lease"

  # Main disk volume using Alpine minirootfs
  volume {
    alias = "artifact"

    pool = "default"
    name = local.vm_name

    source {
      type = "external"
      urls = ["https://dl-cdn.alpinelinux.org/alpine/v3.22/releases/aarch64/alpine-minirootfs-3.22.1-aarch64.tar.gz"]
      checksum = "sha256:188416d41f9f0c9a6e9427b75149e43ccf3a89587b2d27c9ad506e7ffca78d1c"
    }

    capacity = "4G"
    target_dev = "vda"
    bus = "virtio"
    format = "qcow2"
  }

  # Cloud-init for initial setup
  volume {
    source {
      type = "cloud-init"
      user_data = format("#cloud-config\n%s", jsonencode({
        # Set root password
        chpasswd = {
          list = "root:viper"
          expire = false
        }
        # Enable SSH
        ssh_pwauth = true
        # Install basic packages
        packages = [
          "openssh-server",
          "bash",
          "curl"
        ]
        runcmd = [
          ["rc-update", "add", "sshd", "default"],
          ["rc-service", "sshd", "start"]
        ]
      }))
    }
    target_dev = "vdb"
    bus = "virtio"
  }

  shutdown_mode = "acpi"
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