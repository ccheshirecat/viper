#!/bin/sh
set -eux

# This script is downloaded and run by the Packer boot_command.
# The ROOT_PASSWORD variable is exported by the boot_command.

echo "==> Starting Alpine setup script"

# 1. Basic Setup (Network, Timezone, Repos)
setup-hostname -n alpine-template
setup-interfaces -i 'auto lo\niface lo inet loopback\n\nauto eth0\niface eth0 inet dhcp'
rc-update add networking boot
setup-timezone -z UTC
setup-ntp chrony
setup-apkrepos -1

# 2. Set the root password non-interactively.
#    Packer will use this password to SSH into the machine for provisioning.
echo "==> Setting root password"
echo "root:${ROOT_PASSWORD}" | chpasswd

# 3. Configure and start SSH
echo "==> Configuring SSH"
setup-sshd -c openssh
# IMPORTANT: Explicitly permit root login with a password for the provisioning step
sed -i -e 's/^#?PermitRootLogin.*/PermitRootLogin yes/' -e 's/^#?PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config

# 4. Install Alpine to disk. This must be one of the last steps.
echo "==> Installing Alpine to disk"
setup-disk -m sys /dev/vda

echo "==> Alpine setup complete. The system will now reboot."