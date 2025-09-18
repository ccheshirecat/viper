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
  http_content = {
    "/latest/user_data" = <<EOF
echo running packer > logfile.txt
EOF
  }
  machine_type     = "virt"
  ssh_username     = "root"
  ssh_password     = var.ssh_password
  ssh_timeout      = "5m"
  vm_name          = "vipervm"
  vnc_bind_address = "0.0.0.0"
  vnc_use_password = true
  net_device       = "virtio-net"
  disk_interface   = "virtio"
  boot_wait        = "10s"

  # --- Automated Installation using Your Working Sequence ---
  boot_command = [
    "<wait>root<enter>",
    "<wait5>",

    # Create the answer file using your confirmed working sequence
    "cat <<EOF > /tmp/answers.txt \n",
    # Using official ...OPTS variables from the Alpine documentation
    "HOSTNAMEOPTS=alpine \n",
    "INTERFACESOPTS=\"auto lo\\niface lo inet loopback\\n\\nauto eth0\\niface eth0 inet dhcp\" \n",
    "TIMEZONEOPTS=UTC \n",
    "PROXYOPTS=none \n",
    "NTPOPTS=chrony \n",
    "SSHDOPTS=openssh \n",
    "APKREPOSOPTS=\"-1\" \n", # Use first mirror (CDN)
    "DISKOPTS=\"-m sys /dev/vda\" \n",

    # **CRITICAL**: Keeping your non-standard but working password key
    "KEY_ROOT_PASSWORD=${var.ssh_password} \n",
    "EOF",
    "<enter>",

    # **CRITICAL**: Using your confirmed working command and flag
    "<wait>setup-alpine -f /tmp/answers.txt<enter>",

    # Wait for install and reboot
    "<wait1m>reboot<enter>"
  ]
}

# ----------------------
# Build
# ----------------------
build {
  name    = "viper-alpine-rootfs"
  sources = ["source.qemu.alpine"]

  provisioner "shell-local" {
    inline = [
      "echo 'Verifying viper-agent binary exists...'",
      "if [ ! -f '../bin/viper-agent' ]; then",
      "  echo 'ERROR: viper-agent not found at ../bin/viper-agent'",
      "  exit 1",
      "fi"
    ]
  }
  provisioner "shell" {
    inline = [
      "echo 'Updating Alpine and installing base packages...'",
      "apk update",
      "apk add --no-cache bash curl wget openssh ca-certificates tzdata chromium-browser"
    ]
  }
  provisioner "file" {
    source      = "../bin/viper-agent"
    destination = "/usr/local/bin/viper-agent"
  }
  provisioner "shell" {
    inline = [
      "chmod +x /usr/local/bin/viper-agent",
      "echo 'viper-agent installed.'"
    ]
  }
}