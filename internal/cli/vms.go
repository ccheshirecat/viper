package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
	"github.com/ccheshirecat/viper/internal/types"
	"github.com/spf13/cobra"
)

func vmCmd() *cobra.Command {
	vm := &cobra.Command{
		Use:   "vms",
		Short: "Manage microVMs",
		Long:  "Create, list, and destroy microVMs for browser automation tasks.",
	}

	vm.AddCommand(vmCreateCmd())
	vm.AddCommand(vmListCmd())
	vm.AddCommand(vmDestroyCmd())

	return vm
}

func vmCreateCmd() *cobra.Command {
	var (
		memory      int
		cpu         int
		networkMode string
		staticIP    string
		imagePath   string
		bridge      string
		datacenter  string
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new microVM with Cloud Hypervisor",
		Long: `Create and start a new microVM using nomad-driver-ch.

The VM will boot using kernel + initramfs from the Docker build pipeline.
Browser automation is available immediately via the built-in viper-agent.

Examples:
  # Create VM with private subnet networking (default)
  viper vms create my-vm

  # Create VM with static IP
  viper vms create my-vm --network-mode static --static-ip 192.168.1.150

  # Create VM with custom resources
  viper vms create my-vm --memory 2048 --cpu 2000`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			// Parse network mode
			var netMode types.NetworkMode
			switch networkMode {
			case "private", "private_subnet":
				netMode = types.NetworkModePrivateSubnet
			case "static", "static_ip":
				netMode = types.NetworkModeStaticIP
			case "host", "host_shared":
				netMode = types.NetworkModeHostShared
			default:
				netMode = types.NetworkModePrivateSubnet
			}

			// Resolve image paths
			if imagePath == "" {
				imagePath = "./dist" // Default to local build output
			}
			imagePaths := nomad.ResolveImagePaths(imagePath)

			// Validate required files exist
			if err := validateImageFiles(imagePaths); err != nil {
				fmt.Printf("Error: %v\n", err)
				fmt.Println("\nRun 'make build-images' to create VM images first.")
				os.Exit(1)
			}

			// Create job generator
			generator := nomad.NewVMJobGenerator(datacenter, bridge, imagePaths)

			// Generate job
			opts := nomad.VMCreateOptions{
				Name:        name,
				Memory:      memory,
				CPU:         cpu,
				NetworkMode: netMode,
				StaticIP:    staticIP,
				ImagePaths:  imagePaths,
			}

			job, err := generator.GenerateVMJob(opts)
			checkError(err)

			// Deploy via Nomad API
			client, err := nomad.NewClient()
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			jobID, err := client.SubmitJob(ctx, job)
			checkError(err)

			fmt.Printf("✅ VM '%s' deployed successfully\n", name)
			fmt.Printf("Job ID: %s\n", jobID)
			fmt.Printf("Network Mode: %s\n", netMode)
			if netMode == types.NetworkModeStaticIP && staticIP != "" {
				fmt.Printf("Static IP: %s\n", staticIP)
			}
			fmt.Println("\nWait a moment for the VM to boot, then check status:")
			fmt.Printf("  nomad job status %s\n", name)
			fmt.Printf("  viper browsers spawn %s ctx1\n", name)
		},
	}

	// Add flags
	cmd.Flags().IntVar(&memory, "memory", 1024, "Memory allocation in MB")
	cmd.Flags().IntVar(&cpu, "cpu", 1000, "CPU allocation (1000 = 1 core)")
	cmd.Flags().StringVar(&networkMode, "network-mode", "private", "Network mode: private, static, host")
	cmd.Flags().StringVar(&staticIP, "static-ip", "", "Static IP address (for static mode)")
	cmd.Flags().StringVar(&imagePath, "image-path", "./dist", "Path to VM images (kernel, initramfs)")
	cmd.Flags().StringVar(&bridge, "bridge", "br0", "Bridge name for VM networking")
	cmd.Flags().StringVar(&datacenter, "datacenter", "dc1", "Nomad datacenter")

	return cmd
}

// validateImageFiles checks that required VM image files exist
func validateImageFiles(paths nomad.ImagePaths) error {
	if _, err := os.Stat(paths.Kernel); os.IsNotExist(err) {
		return fmt.Errorf("kernel not found: %s", paths.Kernel)
	}
	if _, err := os.Stat(paths.Initramfs); os.IsNotExist(err) {
		return fmt.Errorf("initramfs not found: %s", paths.Initramfs)
	}
	// Disk image is optional
	return nil
}

func vmListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all managed VMs",
		Long:  "Display status information for all managed microVMs.",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := nomad.NewClient()
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			vms, err := client.ListVMs(ctx)
			checkError(err)

			if len(vms) == 0 {
				fmt.Println("No VMs found")
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATUS\tHEALTH\tAGENT\tCONTEXTS\tCREATED")

			for _, vm := range vms {
				contexts := fmt.Sprintf("%d", len(vm.Contexts))
				created := vm.Created.Format("2006-01-02 15:04")
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					vm.Name, vm.Status, vm.Health, vm.AgentURL, contexts, created)
			}

			w.Flush()
		},
	}

	return cmd
}

func vmDestroyCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "destroy [name]",
		Short: "Destroy a microVM",
		Long:  "Stop and remove a microVM and all its resources.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			if !force {
				fmt.Printf("Are you sure you want to destroy VM '%s'? This action cannot be undone. (y/N): ", name)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Cancelled")
					return
				}
			}

			client, err := nomad.NewClient()
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = client.DestroyVM(ctx, name)
			checkError(err)

			fmt.Printf("VM %s destroyed successfully\n", name)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force destruction without confirmation")

	return cmd
}
