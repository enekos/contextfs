package cmd

import (
	"log/slog"
	"mairu/internal/web"
	"os"

	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the Mairu web interface",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		meiliURL, _ := cmd.Flags().GetString("meili-url")
		meiliAPIKey, _ := cmd.Flags().GetString("meili-api-key")
		apiKey := GetAPIKey()
		if apiKey == "" {
			slog.Error("Gemini API key not found. Please run 'mairu setup' or set GEMINI_API_KEY environment variable.")
			os.Exit(1)
		}
		slog.Info("Starting Mairu web interface", "port", port)
		if err := web.StartServer(port, apiKey, meiliURL, meiliAPIKey); err != nil {
			slog.Error("Error starting web server", "error", err)
		}
	},
}

func init() {
	webCmd.Flags().IntP("port", "p", 8080, "Port to run the web server on")
	webCmd.Flags().String("meili-url", os.Getenv("MEILI_URL"), "Meilisearch URL")
	webCmd.Flags().String("meili-api-key", os.Getenv("MEILI_API_KEY"), "Meilisearch API key")
	rootCmd.AddCommand(webCmd)
}
