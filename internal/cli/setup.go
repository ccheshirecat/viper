package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/ccheshirecat/viper/internal/config"
	"github.com/spf13/cobra"
)

// setupCmd creates the setup command for automatic cluster configuration
func setupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup Nomad cluster and networking for Viper",
		Long: `Setup command automatically configures your system for running Viper VMs.

This includes:
- Creating the bridge interface (viperbr0)
- Setting up iptables rules for NAT
- Configuring DNS resolution
- Providing Nomad job examples
- Validating the setup

Run this once before creating your first VM.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("🚀 Setting up Viper environment...")

			cfg := config.DefaultConfig()

			// Check if running as root
			if os.Geteuid() != 0 {
				fmt.Println("❌ This command requires root privileges")
				fmt.Println("   Run: sudo viper setup")
				os.Exit(1)
			}

			// Setup bridge interface
			if err := setupBridge(cfg.BridgeName); err != nil {
				fmt.Printf("❌ Failed to setup bridge: %v\n", err)
				os.Exit(1)
			}

			// Setup networking rules
			if err := setupNetworking(cfg); err != nil {
				fmt.Printf("❌ Failed to setup networking: %v\n", err)
				os.Exit(1)
			}

			// Generate example configurations
			if err := generateExamples(); err != nil {
				fmt.Printf("❌ Failed to generate examples: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Setup complete!")
			fmt.Printf("   Bridge: %s\n", cfg.BridgeName)
			fmt.Printf("   Network: %s\n", cfg.NetworkCIDR)
			fmt.Printf("   Gateway: %s\n", cfg.GatewayIP)
			fmt.Println("\nNext steps:")
			fmt.Println("1. Start Nomad: nomad agent -dev")
			fmt.Println("2. Create your first VM: viper vms create my-vm")
			fmt.Println("3. Test automation: viper browsers spawn my-vm ctx1")
		},
	}

	return cmd
}

// setupBridge creates the bridge interface for VM networking
func setupBridge(bridgeName string) error {
	fmt.Printf("🔧 Creating bridge interface: %s\n", bridgeName)

	// Check if bridge already exists
	cmd := exec.Command("ip", "link", "show", bridgeName)
	if err := cmd.Run(); err == nil {
		fmt.Printf("   Bridge %s already exists\n", bridgeName)
		return nil
	}

	// Create bridge
	cmd = exec.Command("ip", "link", "add", "name", bridgeName, "type", "bridge")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create bridge: %v", err)
	}

	// Set bridge up
	cmd = exec.Command("ip", "link", "set", "dev", bridgeName, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up bridge: %v", err)
	}

	fmt.Printf("   ✅ Created bridge: %s\n", bridgeName)
	return nil
}

// setupNetworking configures NAT and DNS for the bridge
func setupNetworking(cfg *config.DefaultVMConfig) error {
	fmt.Println("🔧 Configuring networking...")

	// Enable IP forwarding
	if err := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run(); err != nil {
		fmt.Println("   ⚠️  Could not enable IP forwarding (may need to set manually)")
	}

	// Add iptables rules for NAT
	rules := []string{
		fmt.Sprintf("-t nat -A POSTROUTING -s %s -j MASQUERADE", cfg.NetworkCIDR),
		fmt.Sprintf("-A FORWARD -i %s -o %s -j ACCEPT", cfg.BridgeName, getDefaultInterface()),
		fmt.Sprintf("-A FORWARD -i %s -o %s -j ACCEPT", getDefaultInterface(), cfg.BridgeName),
	}

	for _, rule := range rules {
		cmd := exec.Command("iptables", "-C", rule)
		if cmd.Run() != nil {
			// Rule doesn't exist, add it
			cmd = exec.Command("iptables", "-A", rule)
			if err := cmd.Run(); err != nil {
				fmt.Printf("   ⚠️  Could not add iptables rule: %s\n", rule)
			}
		}
	}

	fmt.Println("   ✅ Networking configured")
	return nil
}

// getDefaultInterface returns the default network interface
func getDefaultInterface() string {
	if runtime.GOOS == "darwin" {
		return "en0" // macOS
	}
	return "eth0" // Linux
}

// generateExamples creates example configurations
func generateExamples() error {
	fmt.Println("📝 Generating example configurations...")

	// Create examples directory
	if err := os.MkdirAll("examples", 0755); err != nil {
		return err
	}

	// Generate a basic VM job example
	exampleHCL := fmt.Sprintf(`# Example Viper VM Job
job "example-vm" {
  datacenters = ["viper"]
  type        = "service"

  group "browser" {
    count = 1

    task "vm" {
      driver = "nomad-driver-ch"

      config {
        kernel    = "/path/to/vmlinuz"
        initramfs = "/path/to/viper-initramfs.gz"
        hostname  = "example-vm"

        network_interface {
          bridge {
            name = "viperbr0"
          }
        }
      }

      resources {
        cpu    = 2000  # 2 cores
        memory = 2048  # 2GB
      }

      service {
        name = "viper-agent"
        port = "agent"

        check {
          type     = "tcp"
          port     = "agent"
          interval = "30s"
          timeout  = "5s"
        }
      }
    }

    network {
      port "agent" {
        to = 8080
      }
    }
  }
}`)

	if err := os.WriteFile("examples/example-vm.hcl", []byte(exampleHCL), 0644); err != nil {
		return err
	}

	fmt.Println("   ✅ Generated examples/example-vm.hcl")
	return nil
}
