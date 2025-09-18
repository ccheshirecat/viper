packer {
  required_version = ">= 1.9.0"
  required_plugins {
    qemu = {
      version = ">= 1.0.10"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

# ----------------------
# Variables
# ----------------------
variable "version" {
  type    = string
  default = "v0.1.0"
}

variable "output_dir" {
  type    = string
  default = "out"
}

variable "disk_size" {
  type    = number
  default = 2048
}

variable "memory" {
  type    = number
  default = 1024
}

variable "cpu_count" {
  type    = number
  default = 2
}

variable "alpine_version" {
  type    = string
  default = "3.22"
}

variable "ssh_user" {
  type    = string
  default = "root"
}

variable "ssh_key_path" {
  type    = string
  default = "/Users/marcxavier/.ssh/id_rsa"
}

# ----------------------
# Local values
# ----------------------
locals {
  output_filename    = "viper-rootfs-${var.version}"
  alpine_iso_url     = "https://dl-cdn.alpinelinux.org/alpine/v${var.alpine_version}/releases/aarch64/alpine-standard-${var.alpine_version}.1-aarch64.iso"
  alpine_iso_checksum = "4cf7cd3bad64122a8a2423e78369a486a02334d4d88645aab9d08120bb76b0f9"
}

# ----------------------
# QEMU Builder (ARM64 macOS)
# ----------------------
source "qemu" "alpine" {
  qemu_binary       = "/opt/homebrew/bin/qemu-system-aarch64"
  vm_name           = local.output_filename
  output_directory  = "${var.output_dir}/${local.output_filename}"

  # Disk/Image
  format            = "qcow2"
  disk_size         = var.disk_size
  disk_interface    = "virtio"

  # Use Alpine cloud image instead of ISO
  disk_image        = true
  iso_url           = "https://dl-cdn.alpinelinux.org/alpine/v3.22/releases/cloud/generic_alpine-3.22.1-aarch64-uefi-cloudinit-r0.qcow2"
  iso_checksum      = "sha512:5cf7697f03e4b5280c25c86d910e0691c1f1210787f3276b458d4c183477c31e551e9c0bd70e8a4d87526c47668bc1b582a6fac4bf3bbd5c6ea844f36eab111a"

  # Hardware
  memory            = var.memory
  cpus              = var.cpu_count
  accelerator       = "hvf"
  net_device        = "virtio-net"

  # Headless
  headless          = true
  use_default_display = false

  # SSH
  communicator      = "ssh"
  ssh_username      = "root"
  ssh_password      = "viper"
  ssh_timeout       = "20m"

  # Cloud image boots directly - no boot commands needed
  boot_wait         = "60s"

  shutdown_command  = "poweroff"
  shutdown_timeout  = "5m"
  # Cloud-init configuration for SSH setup
  cd_files         = ["../meta-data", "../user-data"]
  cd_label         = "cidata"

  qemuargs = [
    ["-machine", "virt"],
    ["-boot", "strict=off"]
  ]
}

# ----------------------
# Build
# ----------------------
build {
  name    = "viper-alpine-rootfs"
  sources = ["source.qemu.alpine"]


  # Verify viper-agent binary locally
  provisioner "shell-local" {
    inline = [
      "echo 'Verifying viper-agent binary exists...'",
      "if [ ! -f '../bin/viper-agent' ]; then",
      "  echo 'ERROR: viper-agent not found at ../bin/viper-agent'",
      "  exit 1",
      "fi"
    ]
  }

  # Base system setup inside VM
  provisioner "shell" {
    inline = [
      "echo 'Updating Alpine and installing base packages...'",
      "apk update",
      "apk add --no-cache bash curl wget openssh ca-certificates tzdata"
    ]
  }

  # Copy viper-agent into VM
  provisioner "file" {
    source      = "../bin/viper-agent"
    destination = "/usr/local/bin/viper-agent"
  }

  # Make viper-agent executable
  provisioner "shell" {
    inline = [
      "chmod +x /usr/local/bin/viper-agent",
      "echo 'viper-agent installed.'"
    ]
  }
}