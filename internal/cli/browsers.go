package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/viper-org/viper/pkg/client"
)

func browserCmd() *cobra.Command {
	browsers := &cobra.Command{
		Use:   "browsers",
		Short: "Manage browser contexts",
		Long:  "Spawn and manage isolated browser contexts within microVMs.",
	}

	browsers.AddCommand(browserSpawnCmd())
	browsers.AddCommand(browserListCmd())
	browsers.AddCommand(browserDestroyCmd())

	return browsers
}

func browserSpawnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn [vm] [context-id]",
		Short: "Spawn a new browser context",
		Long:  "Create a new isolated browser context within the specified microVM.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			contextID := args[1]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = agentClient.SpawnContext(ctx, contextID)
			checkError(err)

			fmt.Printf("Browser context '%s' spawned successfully in VM '%s'\n", contextID, vmName)
		},
	}

	return cmd
}

func browserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [vm]",
		Short: "List browser contexts",
		Long:  "Display all active browser contexts in the specified microVM.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			contexts, err := agentClient.ListContexts(ctx)
			checkError(err)

			if len(contexts) == 0 {
				fmt.Printf("No browser contexts found in VM '%s'\n", vmName)
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "CONTEXT ID\tACTIVE\tPROFILE\tCREATED\tLAST USED")

			for _, context := range contexts {
				active := "No"
				if context.Active {
					active = "Yes"
				}

				profileName := "None"
				if context.Profile != nil && context.Profile.Name != "" {
					profileName = context.Profile.Name
				}

				created := context.Created.Format("2006-01-02 15:04")

				lastUsed := "Never"
				if context.LastUsed != nil {
					lastUsed = context.LastUsed.Format("2006-01-02 15:04")
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					context.ID, active, profileName, created, lastUsed)
			}

			w.Flush()
		},
	}

	return cmd
}

func browserDestroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy [vm] [context-id]",
		Short: "Destroy a browser context",
		Long:  "Terminate and remove a browser context from the specified microVM.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			contextID := args[1]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = agentClient.DestroyContext(ctx, contextID)
			checkError(err)

			fmt.Printf("Browser context '%s' destroyed successfully in VM '%s'\n", contextID, vmName)
		},
	}

	return cmd
}