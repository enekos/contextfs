package cmd

import (
	"context"
	"log/slog"
	"mairu/internal/contextsrv"
	"mairu/internal/logger"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var contextServerCmd = &cobra.Command{
	Use:   "context-server",
	Short: "Start centralized context management server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		pgDSN, _ := cmd.Flags().GetString("pg-dsn")
		authToken, _ := cmd.Flags().GetString("auth-token")
		enableProjector, _ := cmd.Flags().GetBool("projector")
		meiliURL, _ := cmd.Flags().GetString("meili-url")
		meiliAPIKey, _ := cmd.Flags().GetString("meili-api-key")

		if pgDSN == "" {
			pgDSN = os.Getenv("CONTEXT_SERVER_POSTGRES_DSN")
		}

		cfg := contextsrv.LoadConfig()
		cfg.Port = port
		cfg.PostgresDSN = pgDSN
		if meiliURL != "" {
			cfg.MeiliURL = meiliURL
		}
		if meiliAPIKey != "" {
			cfg.MeiliAPIKey = meiliAPIKey
		}
		if authToken != "" {
			cfg.AuthToken = authToken
		}
		cfg.EnableProjector = enableProjector

		logger.Init(logger.Config{
			Level:      "info",
			Structured: true,
		})

		app, err := contextsrv.NewApp(cfg)
		if err != nil {
			slog.Error("failed to initialize context server", "error", err)
			os.Exit(1)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		go func() {
			<-ctx.Done()
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			_ = app.Shutdown(shutdownCtx)
		}()

		slog.Info("Starting context server", "port", port)
		if err := app.Start(ctx); err != nil && err.Error() != "http: Server closed" {
			slog.Error("context server exited with error", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	contextServerCmd.Flags().IntP("port", "p", 8788, "Port to run context server on")
	contextServerCmd.Flags().String("pg-dsn", os.Getenv("CONTEXT_SERVER_POSTGRES_DSN"), "PostgreSQL DSN")
	contextServerCmd.Flags().String("auth-token", os.Getenv("CONTEXT_SERVER_AUTH_TOKEN"), "Shared token for internal auth (X-Context-Token)")
	contextServerCmd.Flags().Bool("projector", true, "Enable Postgres->Meilisearch outbox projector")
	contextServerCmd.Flags().String("meili-url", os.Getenv("MEILI_URL"), "Meilisearch URL")
	contextServerCmd.Flags().String("meili-api-key", os.Getenv("MEILI_API_KEY"), "Meilisearch API key")
	rootCmd.AddCommand(contextServerCmd)
}
