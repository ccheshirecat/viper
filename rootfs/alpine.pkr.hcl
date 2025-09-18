packer {
  required_version = ">= 1.9.0"
  required_plugins {
    qemu = {
      version = ">= 1.0.10"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

source "qemu" "alpine" {
  qemuargs = [

    ["-machine", "virt"],
  # Arbitrary QEMU args (for ARM64 HVF, ISO boot)
    ["-boot", "strict=off"],
  ]
  qemu_binary       = "/opt/homebrew/bin/qemu-system-aarch64"
  iso_url          = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/aarch64/alpine-virt-3.21.1-aarch64.iso"
  iso_checksum     = "sha256:c6b72a153782d4043c0719a196f8bb8e749f2e8027ca4000579866729a312697"
  shutdown_command = "echo 'packer' | poweroff"
  disk_size        = "10000M"
  format           = "qcow2"
  accelerator      = "hvf"
  headless         = true
  http_content = {
    "/latest/user_data" = <<EOF
echo running packer > logfile.txt
EOF
  }
  machine_type     = "virt"
  ssh_username     = "root"
  ssh_password     = "password"
  ssh_timeout      = "5m"
  vm_name          = "vipervm"
  vnc_bind_address = "0.0.0.0"
  vnc_use_password = true
  net_device       = "virtio-net"
  disk_interface   = "virtio"
  boot_wait        = "10s"
  boot_command     = ["<wait>root<enter><wait>setup-alpine<enter><enter>alpine<enter>eth0<enter>dhcp<enter>n<enter><wait10>password<enter>password<enter>UTC<enter><wait10><enter>r<wait5><enter>no<enter><wait><wait5>openssh<enter><wait5>yes<enter>none<enter>vda<enter>sys<enter>y<enter><wait2m><enter><wait10>"]
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