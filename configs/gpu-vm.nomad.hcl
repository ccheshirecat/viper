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

    task "agent" {
      driver = "exec"

      config {
        command = "/usr/local/bin/viper-agent"
        args = [
          "--listen=:8080",
          "--vm-name=gpu-example",
          "--task-dir=/var/viper/tasks",
          "--gpu-enabled"
        ]
      }

      resources {
        cpu    = 4000  # 4 CPU cores for GPU workloads
        memory = 8192  # 8GB RAM for GPU workloads

        # GPU device request
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
            static = 8080
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