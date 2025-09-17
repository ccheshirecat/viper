package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "viper",
	Short: "Viper: microVM + browser automation CLI",
	Long: `Viper is a production-grade microVM-based browser automation framework
that provides unparalleled session persistence, kernel-level security,
and massive scalability for stateful browser tasks where stealth,
reliability, and performance are paramount.`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(vmCmd())
	rootCmd.AddCommand(taskCmd())
	rootCmd.AddCommand(browserCmd())
	rootCmd.AddCommand(profileCmd())
	rootCmd.AddCommand(debugCmd())
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}