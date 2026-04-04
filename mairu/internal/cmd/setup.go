package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup Mairu configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Mairu Setup!")
		fmt.Print("Please enter your Gemini API Key: ")
		reader := bufio.NewReader(os.Stdin)
		apiKey, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			os.Exit(1)
		}
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			fmt.Println("API Key cannot be empty.")
			os.Exit(1)
		}

		cfg, err := LoadConfig()
		if err != nil {
			cfg = Config{}
		}
		cfg.GeminiAPIKey = apiKey

		if err := SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Configuration saved successfully to ~/.config/mairu/config.json!")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
