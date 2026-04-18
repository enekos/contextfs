//go:build contextsrvonly

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"mairu/internal/cmd/admincmd"
)

func init() {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println("Mairu context-server build. Use 'mairu context-server' to start.")
		cmd.Help()
	}

	rootCmd.AddCommand(
		NewContextServerCmd(),
		admincmd.NewCompletionCmd(rootCmd),
		NewDoctorCmd(),
		NewSetupCmd(),
		NewConfigCmd(),
	)
}
