job "viper-vm-example" {
  datacenters = ["dc1"]
  type        = "service"

  group "vm-group" {
    count = 1

    task "microvm" {
      driver = "virt"

      config {
        # VM Image - our Alpine rootfs with viper-agent
        image = "file:///Users/marcxavier/Desktop/viper/out/cloudhypervisor-rootfs/viper-rootfs-latest.qcow2"

        # VM Resources
        memory = "2048"   # 2GB RAM
        vcpu   = 2        # 2 CPU cores

        # Network Configuration
        network_interface {
          type   = "bridge"
          source = "virbr0"  # Default libvirt bridge
          model  = "virtio"  # High-performance network
        }

        # Boot Configuration
        boot {
          loader_type = "bios"
        }

        # Console access for debugging
        console {
          type = "pty"
        }
      }

      resources {
        # Host resources allocated to this VM
        cpu    = 2000  # 2 CPU cores worth of host CPU
        memory = 2560  # 2.5GB host memory (VM memory + overhead)

        network {
          mbits = 1000  # High bandwidth for browser automation
          port "http" {
            to = 8080     # Agent port inside VM
          }
        }
      }

      # Service discovery for the agent inside the VM
      service {
        name = "viper-agent-example"
        port = "http"

        check {
          type     = "tcp"      # TCP check since HTTP needs VM to be fully booted
          port     = "http"
          interval = "30s"      # Longer interval for VM startup
          timeout  = "10s"
        }
      }
    }
  }

  # Restart policy - more conservative for VMs
  reschedule {
    delay          = "60s"       # Longer delay for VM restart
    delay_function = "exponential"
    max_delay      = "300s"      # 5 minutes max
    unlimited      = true
  }

  # Update strategy - careful with VM updates
  update {
    max_parallel     = 1         # One VM at a time
    min_healthy_time = "60s"     # Wait for VM to fully boot
    healthy_deadline = "5m"      # VMs take longer to start
    auto_revert      = true
  }
}