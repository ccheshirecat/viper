package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
	"github.com/ccheshirecat/viper/pkg/client"
	"github.com/spf13/cobra"
)

func profileCmd() *cobra.Command {
	profiles := &cobra.Command{
		Use:   "profiles",
		Short: "Manage browser profiles",
		Long:  "Attach browser profiles to contexts for session persistence and customization.",
	}

	profiles.AddCommand(profileAttachCmd())

	return profiles
}

func profileAttachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach [vm] [context-id] [profile.json]",
		Short: "Attach profile to browser context",
		Long:  "Load and apply a browser profile configuration to the specified context.",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			contextID := args[1]
			profileFile := args[2]

			data, err := ioutil.ReadFile(profileFile)
			checkError(err)

			var profile types.Profile
			err = json.Unmarshal(data, &profile)
			checkError(err)

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = agentClient.AttachProfile(ctx, contextID, profile)
			checkError(err)

			fmt.Printf("Profile '%s' attached successfully to context '%s' in VM '%s'\n",
				profile.Name, contextID, vmName)

			if profile.UserAgent != "" {
				fmt.Printf("  User Agent: %s\n", profile.UserAgent)
			}
			if len(profile.Cookies) > 0 {
				fmt.Printf("  Cookies: %d loaded\n", len(profile.Cookies))
			}
			if len(profile.LocalStorage) > 0 {
				fmt.Printf("  LocalStorage: %d domains configured\n", len(profile.LocalStorage))
			}
		},
	}

	return cmd
}