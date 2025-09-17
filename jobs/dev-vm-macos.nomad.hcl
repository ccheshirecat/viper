job "viper-vm-dev-macos" {
  datacenters = ["dc1"]
  type        = "service"

  # This template is for macOS development testing only
  # Production Linux environments should use the virt driver template

  group "vm-group" {
    count = 1

    # Modern network configuration (Nomad 0.12+)
    network {
      port "http" {
        # Agent port forwarded from VM
      }
    }

    task "qemu-vm" {
      driver = "exec"

      config {
        # Start QEMU directly with our Alpine rootfs
        command = "/opt/homebrew/bin/qemu-system-x86_64"
        args = [
          "-m", "1024",                    # 1GB RAM
          "-smp", "2",                     # 2 CPU cores
          "-drive", "file=${NOMAD_TASK_DIR}/viper-rootfs.qcow2,format=qcow2,if=virtio",
          "-netdev", "user,id=net0,hostfwd=tcp::${NOMAD_PORT_http}-:8080",
          "-device", "virtio-net-pci,netdev=net0",
          "-nographic",                    # No GUI for headless operation
          "-daemonize",                   # Run in background
          "-pidfile", "${NOMAD_TASK_DIR}/qemu.pid"
        ]
      }

      # Copy the rootfs image to the task directory
      artifact {
        source      = "file:///Users/marcxavier/Desktop/viper/out/viper-rootfs-latest.qcow2"
        destination = "local/viper-rootfs.qcow2"
        mode        = "file"
      }

      resources {
        cpu    = 2000  # 2 CPU cores
        memory = 1536  # 1.5GB host memory (VM + overhead)
      }

      # Service discovery for the agent inside the VM
      service {
        name = "viper-agent-dev"
        port = "http"

        check {
          type     = "tcp"
          port     = "http"
          interval = "30s"
          timeout  = "10s"
        }
      }

      # Kill QEMU gracefully
      kill_signal = "SIGTERM"
      kill_timeout = "30s"

      # Template for shutdown script
      template {
        data = <<-EOF
#!/bin/bash
if [ -f ${NOMAD_TASK_DIR}/qemu.pid ]; then
    kill -TERM $(cat ${NOMAD_TASK_DIR}/qemu.pid) 2>/dev/null || true
    rm -f ${NOMAD_TASK_DIR}/qemu.pid
fi
EOF
        destination = "local/shutdown.sh"
        perms       = "755"
      }
    }
  }

  # Restart policy
  reschedule {
    delay          = "30s"
    delay_function = "exponential"
    max_delay      = "120s"
    unlimited      = true
  }

  # Update strategy
  update {
    max_parallel     = 1
    min_healthy_time = "60s"   # Wait for VM to boot and agent to start
    healthy_deadline = "3m"
    auto_revert      = true
  }
}