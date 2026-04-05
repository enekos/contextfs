package cmd

import (
	"fmt"
	"log/slog"
	"mairu/internal/agent"
	"mairu/internal/logger"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var debugMode bool

var rootCmd = &cobra.Command{
	Use:   "mairu [prompt]",
	Short: "Mairu - The coding agent that knows your codebase",
	Long:  `Mairu is a graph-powered AI coding agent built for performance and deep context.`,
	Args:  cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Setup(debugMode)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			prompt := strings.Join(args, " ")
			runHeadless(prompt)
			return
		}
		fmt.Println("Welcome to Mairu! Use 'mairu tui' or 'mairu web' to start.")
		cmd.Help()
	},
}

func runHeadless(prompt string) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		slog.Error("Gemini API key not found. Please run 'mairu setup' or set GEMINI_API_KEY environment variable.")
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	a, err := agent.New(cwd, apiKey)
	if err != nil {
		slog.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}
	defer a.Close()

	response, err := a.Run(prompt)
	if err != nil {
		slog.Error("Agent error", "error", err)
		os.Exit(1)
	}

	fmt.Println("\n" + response)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
}
