package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
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
		memory     int
		cpu        int
		imagePath  string
		datacenter string
		debug      bool
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new microVM with Cloud Hypervisor",
		Long: `Create and start a new microVM using nomad-driver-ch.

The VM will boot using kernel + initramfs and automatically get:
- Private network with automatic IP assignment
- Service discovery for browser automation
- Default resources (2GB RAM, 2 CPU cores)

Run 'viper setup' first to configure networking automatically.

Examples:
  # Create VM with defaults (recommended)
  viper vms create my-vm

  # Create VM with custom resources
  viper vms create my-vm --memory 4096 --cpu 4000

  # Show configuration without creating
  viper vms create my-vm --debug`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			// Resolve image paths
			if imagePath == "" {
				imagePath = "./dist" // Default to local build output
			}
			imagePaths := nomad.ResolveImagePaths(imagePath)

			// Validate required files exist (skip in debug mode)
			if !debug {
				if err := validateImageFiles(imagePaths); err != nil {
					fmt.Printf("Error: %v\n", err)
					fmt.Println("\nRun 'make build-images' to create VM images first.")
					os.Exit(1)
				}
			} else {
				// Use dummy image paths for debug mode
				imagePaths = nomad.ImagePaths{
					Kernel:    "/path/to/vmlinuz",
					Initramfs: "/path/to/initramfs.gz",
				}
			}

			// Create job generator with opinionated defaults
			generator := nomad.NewVMJobGenerator(datacenter, imagePaths)

			// Generate job with defaults
			opts := nomad.VMCreateOptions{
				Name:       name,
				Memory:     memory,
				CPU:        cpu,
				ImagePaths: imagePaths,
			}

			if debug {
				// Show HCL configuration instead of submitting
				hcl, err := generator.GenerateJobHCL(opts)
				checkError(err)

				fmt.Printf("=== Generated Nomad Job HCL ===\n")
				fmt.Printf("%s\n", hcl)
				fmt.Printf("=== End HCL ===\n")
				return
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

			fmt.Printf("✅ VM '%s' created successfully!\n", name)
			fmt.Printf("Job ID: %s\n", jobID)
			fmt.Println("\nWait a moment for the VM to boot, then check status:")
			fmt.Printf("  nomad job status %s\n", name)
			fmt.Printf("  viper browsers spawn %s ctx1\n", name)
		},
	}

	// Add flags - keep it simple!
	cmd.Flags().IntVar(&memory, "memory", 2048, "Memory allocation in MB (default: 2048)")
	cmd.Flags().IntVar(&cpu, "cpu", 2000, "CPU allocation (1000 = 1 core, default: 2000)")
	cmd.Flags().StringVar(&imagePath, "image-path", "./dist", "Path to VM images (kernel, initramfs)")
	cmd.Flags().StringVar(&datacenter, "datacenter", "viper", "Nomad datacenter")
	cmd.Flags().BoolVar(&debug, "debug", false, "Show job configuration without submitting")

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
