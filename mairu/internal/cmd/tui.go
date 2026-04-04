package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mairu/internal/agent"
	"mairu/internal/tui"
)

var sessionName string

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the Mairu interactive terminal session",
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := GetAPIKey()
		if apiKey == "" {
			fmt.Println("Error: Gemini API key not found. Please run 'mairu setup' or set GEMINI_API_KEY environment variable.")
			os.Exit(1)
		}

		cwd, _ := os.Getwd()
		a, err := agent.New(cwd, apiKey)
		if err != nil {
			fmt.Printf("Failed to initialize agent: %v\n", err)
			os.Exit(1)
		}
		defer a.Close()

		if sessionName != "" {
			if err := a.LoadSession(sessionName); err != nil {
				fmt.Printf("Warning: Failed to load session '%s': %v\n", sessionName, err)
			}
		}

		if err := tui.Start(a, sessionName); err != nil {
			fmt.Printf("TUI Error: %v\n", err)
			os.Exit(1)
		}

		if sessionName != "" {
			a.SaveSession(sessionName)
		}
	},
}

func init() {
	tuiCmd.Flags().StringVarP(&sessionName, "session", "s", "", "Load/Save a named session (e.g. 'bug-123')")
	rootCmd.AddCommand(tuiCmd)
}
