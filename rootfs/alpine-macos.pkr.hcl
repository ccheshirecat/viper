packer {
  required_version = ">= 1.9.0"
  required_plugins {
    qemu = {
      version = ">= 1.0.10"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

variable "ssh_password" {
  type      = string
  default   = "password"
  sensitive = true
}

source "qemu" "alpine" {
  qemuargs = [
    ["-cpu", "host"],
    ["-bios", "/opt/homebrew/share/qemu/edk2-aarch64-code.fd"],
    ["-boot", "strict=off"],
  ]
  qemu_binary      = "/opt/homebrew/bin/qemu-system-aarch64"
  iso_url          = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/aarch64/alpine-virt-3.21.1-aarch64.iso"
  iso_checksum     = "sha256:c6b72a153782d4043c0719a196f8bb8e749f2e8027ca4000579866729a312697"
  shutdown_command = "poweroff"
  disk_size        = "10000M"
  format           = "qcow2"
  accelerator      = "hvf"
  headless         = true
  machine_type     = "virt"

  # Packer connects as root using a TEMPORARY password for provisioning.
  ssh_username     = "root"
  ssh_password     = var.ssh_password
  ssh_timeout      = "20m"

  # This directory contains the setup.sh script.
  http_directory   = "http"

  vm_name          = "vipervm"
  net_device       = "virtio-net"
  disk_interface   = "virtio"
  boot_wait        = "20s"

  # --- Direct, Non-Interactive Installation (based on your link) ---
  boot_command = [
    "<wait10>root<enter><wait5>",
    # Export the password so the setup script can use it
    "export ROOT_PASSWORD='${var.ssh_password}'<enter>",
    # Download, make executable, and run the setup script
    "wget http://{{ .HTTPIP }}:{{ .HTTPPort }}/setup.sh && chmod +x setup.sh && ./setup.sh<enter>",
    # The setup script itself will reboot the machine.
  ]
}

# ----------------------
# Build
# ----------------------
build {
  name    = "viper-alpine-rootfs"
  sources = ["source.qemu.alpine"]

  # 1. Verify viper-agent binary locally (your original provisioner)
  provisioner "shell-local" {
    inline = [
      "echo '==> Verifying viper-agent binary exists...'",
      "if [ ! -f '../bin/viper-agent' ]; then",
      "  echo 'ERROR: viper-agent not found at ../bin/viper-agent'",
      "  exit 1",
      "fi"
    ]
  }

  # 2. Base system setup inside VM
  provisioner "shell" {
    inline = [
      "echo '==> Installing base packages and cloud-init...'",
      "apk update",
      # Combined package list
      "apk add --no-cache cloud-init bash curl wget openssh ca-certificates tzdata"
    ]
  }

  # 3. Configure cloud-init
  provisioner "shell" {
    inline = [
      "echo '==> Configuring cloud-init...'",
      "setup-cloud-init -c /etc/cloud/cloud.cfg -a",
      "echo 'datasource_list: [ NoCloud, ConfigDrive ]' > /etc/cloud/cloud.cfg.d/99_datasource.cfg",
      "rc-update add cloud-init default",
      "rc-update add sshd default",
    ]
  }

  # 4. Copy viper-agent into VM (your original provisioner)
  provisioner "file" {
    source      = "../bin/viper-agent"
    destination = "/tmp/viper-agent"
  }

  # 5. Make viper-agent executable (your original provisioner)
  provisioner "shell" {
    inline = [
      "mv /tmp/viper-agent /usr/local/bin/viper-agent",
      "chmod +x /usr/local/bin/viper-agent",
      "echo '==> viper-agent installed.'"
    ]
  }

  # 6. FINAL CLEANUP: This MUST be the last provisioner
  provisioner "shell" {
    inline = [
      "echo '==> Cleaning up for template image...'",
      # CRITICAL: Remove the temporary root password
      "passwd -d root",
      # CRITICAL: Disable password-based SSH login for security
      "sed -i 's/^PermitRootLogin yes/PermitRootLogin prohibit-password/g' /etc/ssh/sshd_config",
      "sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/g' /etc/ssh/sshd_config",
      # Clean up logs and shell history
      "rm -f /root/.ash_history",
      "logrotate -f /etc/logrotate.conf || true",
      "rm -f /var/log/*-* /var/log/*.gz || true"
    ]
  }
}