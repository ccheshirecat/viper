package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
	"github.com/ccheshirecat/viper/pkg/client"
	"github.com/spf13/cobra"
)

func debugCmd() *cobra.Command {
	debug := &cobra.Command{
		Use:   "debug",
		Short: "Debug and diagnostics",
		Long:  "Diagnostic commands for system health, network connectivity, and agent status.",
	}

	debug.AddCommand(debugSystemCmd())
	debug.AddCommand(debugNetworkCmd())
	debug.AddCommand(debugAgentCmd())

	return debug
}

func debugSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System diagnostics",
		Long:  "Display Nomad cluster health and VM orchestration status.",
		Run: func(cmd *cobra.Command, args []string) {
			nomadClient, err := nomad.NewClient()
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			status, err := nomadClient.GetSystemStatus(ctx)
			checkError(err)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "COMPONENT\tSTATUS\tDETAILS")

			fmt.Fprintf(w, "Nomad Cluster\t%s\t%s\n",
				status.NomadStatus, status.NomadDetails)
			fmt.Fprintf(w, "VM Count\t%d\t%s\n",
				status.VMCount, status.VMDetails)
			fmt.Fprintf(w, "Active Tasks\t%d\t%s\n",
				status.ActiveTasks, status.TaskDetails)

			w.Flush()
		},
	}

	return cmd
}

func debugNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Network diagnostics",
		Long:  "Test network connectivity and proxy configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Network Diagnostics")
			fmt.Println("==================")

			nomadClient, err := nomad.NewClient()
			if err != nil {
				fmt.Printf("✗ Nomad connection: %v\n", err)
			} else {
				fmt.Printf("✓ Nomad connection: OK\n")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			vms, err := nomadClient.ListVMs(ctx)
			if err != nil {
				fmt.Printf("✗ VM list retrieval: %v\n", err)
				return
			}

			fmt.Printf("\nTesting VM Agent Connectivity:\n")
			for _, vm := range vms {
				if vm.AgentURL == "" {
					fmt.Printf("✗ %s: No agent URL\n", vm.Name)
					continue
				}

				agentClient, err := client.NewAgentClient(vm.Name)
				if err != nil {
					fmt.Printf("✗ %s: Client creation failed: %v\n", vm.Name, err)
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				health, err := agentClient.Health(ctx)
				cancel()

				if err != nil {
					fmt.Printf("✗ %s: Health check failed: %v\n", vm.Name, err)
				} else {
					fmt.Printf("✓ %s: %s (uptime: %s)\n", vm.Name, health.Status, health.Uptime)
				}
			}
		},
	}

	return cmd
}

func debugAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [vm]",
		Short: "Agent diagnostics",
		Long:  "Display detailed agent health and performance metrics.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			health, err := agentClient.Health(ctx)
			checkError(err)

			fmt.Printf("Agent Diagnostics for VM: %s\n", vmName)
			fmt.Println("================================")

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Status:\t%s\n", health.Status)
			fmt.Fprintf(w, "Version:\t%s\n", health.Version)
			fmt.Fprintf(w, "Uptime:\t%s\n", health.Uptime)
			fmt.Fprintf(w, "Active Contexts:\t%d\n", health.Contexts)
			fmt.Fprintf(w, "Running Tasks:\t%d\n", health.Tasks)
			fmt.Fprintf(w, "Memory Usage:\t%.2f MB\n", float64(health.Memory)/(1024*1024))
			fmt.Fprintf(w, "Last Check:\t%s\n", health.LastCheck.Format("2006-01-02 15:04:05"))

			if len(health.Details) > 0 {
				fmt.Fprintf(w, "\nAdditional Details:\n")
				for key, value := range health.Details {
					fmt.Fprintf(w, "  %s:\t%s\n", key, value)
				}
			}

			w.Flush()
		},
	}

	return cmd
}
