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
	var config types.VMConfig

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new microVM",
		Long:  "Create and start a new microVM with the specified configuration.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			config.Name = args[0]

			client, err := nomad.NewClient()
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			vmStatus, err := client.CreateVM(ctx, config)
			checkError(err)

			fmt.Printf("VM %s created successfully\n", vmStatus.Name)
			fmt.Printf("Status: %s\n", vmStatus.Status)
			if vmStatus.AgentURL != "" {
				fmt.Printf("Agent URL: %s\n", vmStatus.AgentURL)
			}
		},
	}

	cmd.Flags().StringVar(&config.VMM, "vmm", "cloudhypervisor", "VMM to use (cloudhypervisor, firecracker)")
	cmd.Flags().IntVar(&config.Contexts, "contexts", 1, "Number of browser contexts to support")
	cmd.Flags().BoolVar(&config.GPU, "gpu", false, "Enable GPU passthrough")
	cmd.Flags().IntVar(&config.Memory, "memory", 2048, "Memory allocation in MB")
	cmd.Flags().IntVar(&config.CPUs, "cpus", 2, "Number of CPUs to allocate")
	cmd.Flags().IntVar(&config.Disk, "disk", 8192, "Disk size in MB")

	return cmd
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
