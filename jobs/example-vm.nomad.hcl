job "viper-vm-example" {
  datacenters = ["dc1"]
  type        = "service"

  group "vm-group" {
    count = 1

    task "agent" {
      driver = "exec"

      config {
        command = "/usr/local/bin/viper-agent"
        args = [
          "--listen=:8080",
          "--vm-name=example",
          "--task-dir=/var/viper/tasks"
        ]
      }

      resources {
        cpu    = 2000  # 2 CPU cores
        memory = 2048  # 2GB RAM

        network {
          mbits = 100
          port "http" {
            static = 8080
          }
        }
      }

      # Health check configuration
      service {
        name = "viper-agent-example"
        port = "http"

        check {
          type     = "http"
          path     = "/health"
          interval = "10s"
          timeout  = "3s"
        }
      }

      # Environment variables for the agent
      env {
        GOMAXPROCS = "2"
        GIN_MODE   = "release"
      }

      # Logging configuration
      logs {
        max_files     = 10
        max_file_size = 10
      }

      # Kill signal and timeout
      kill_timeout = "30s"
      kill_signal  = "SIGTERM"
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
    min_healthy_time = "30s"
    healthy_deadline = "3m"
    auto_revert      = true
  }
}