# Viper Rootfs - Production-Ready Alpine Linux Image via libvirt
# This template builds through libvirt abstraction layer for true hypervisor independence

packer {
  required_version = ">= 1.9.0"
  required_plugins {
    shell = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/shell"
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
  alpine_iso = "https://dl-cdn.alpinelinux.org/alpine/v3.22/releases/aarch64/alpine-standard-3.22.1-aarch64.iso"
  output_path = "out/${local.vm_name}.qcow2"
}

# Use shell-local provisioner to drive libvirt
build {
  name = "viper-alpine-libvirt"

  sources = ["source.null.libvirt-build"]

  # Create VM via libvirt
  provisioner "shell-local" {
    inline = [
      "echo 'Creating VM via libvirt...'",
      "mkdir -p out",
      "virt-install \\",
      "  --name ${local.vm_name} \\",
      "  --ram ${var.memory} \\",
      "  --vcpus ${var.cpus} \\",
      "  --disk path=${local.output_path},format=qcow2,size=4 \\",
      "  --cdrom ${local.alpine_iso} \\",
      "  --network network=default \\",
      "  --graphics vnc,listen=127.0.0.1,port=5900 \\",
      "  --noautoconsole \\",
      "  --wait=-1",
      "echo 'VM created via libvirt'"
    ]
  }
}

source "null" "libvirt-build" {
  communicator = "none"
}