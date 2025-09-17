job "viper-vm-gpu-example" {
  datacenters = ["dc1"]
  type        = "service"

  group "vm-group" {
    count = 1

    # GPU constraint
    constraint {
      attribute = "${driver.nvidia.available}"
      operator  = "="
      value     = "true"
    }

    task "microvm" {
      driver = "virt"

      config {
        # VM Image - our Alpine rootfs with viper-agent
        image = "file:///Users/marcxavier/Desktop/viper/out/cloudhypervisor-rootfs/viper-rootfs-latest.qcow2"

        # VM Resources - more for GPU workloads
        memory = "8192"   # 8GB RAM for GPU workloads
        vcpu   = 4        # 4 CPU cores

        # Network Configuration
        network_interface {
          type   = "bridge"
          source = "virbr0"
          model  = "virtio"
        }

        # GPU Passthrough Configuration
        hostdev {
          mode = "subsystem"
          type = "pci"
          managed = "yes"
          source {
            address {
              domain = "0x0000"
              bus    = "0x01"
              slot   = "0x00"
              function = "0x0"
            }
          }
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
        cpu    = 4000  # 4 CPU cores worth of host CPU
        memory = 9216  # 9GB host memory (8GB VM + overhead)

        # Host GPU device allocation (handled at Nomad level)
        device "nvidia/gpu" {
          count = 1

          # Optional: specific GPU constraints
          constraint {
            attribute = "${device.attr.memory}"
            operator  = ">="
            value     = "4096"  # Minimum 4GB VRAM
          }
        }

        network {
          mbits = 1000  # Higher bandwidth for GPU workloads
          port "http" {
            to = 8080     # Agent port inside VM
          }
        }
      }

      service {
        name = "viper-agent-gpu-example"
        port = "http"

        check {
          type     = "http"
          path     = "/health"
          interval = "10s"
          timeout  = "5s"
        }
      }

      # GPU-specific environment variables
      env {
        NVIDIA_VISIBLE_DEVICES = "all"
        GOMAXPROCS            = "4"
        GIN_MODE              = "release"
      }

      logs {
        max_files     = 10
        max_file_size = 10
      }

      kill_timeout = "60s"
      kill_signal  = "SIGTERM"
    }
  }

  reschedule {
    delay          = "60s"
    delay_function = "exponential"
    max_delay      = "300s"
    unlimited      = true
  }

  update {
    max_parallel     = 1
    min_healthy_time = "60s"
    healthy_deadline = "5m"
    auto_revert      = true
  }
}