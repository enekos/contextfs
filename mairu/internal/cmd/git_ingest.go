package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mairu/internal/config"
	"mairu/internal/git"

	"github.com/spf13/cobra"
)

func NewGitIngestCmd() *cobra.Command {
	var sinceStr string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest git history into context nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}

			project := resolveProjectName(dir)

			var since time.Time
			if sinceStr != "" {
				since, err = parseDurationOrDate(sinceStr)
				if err != nil {
					return fmt.Errorf("invalid --since value: %w", err)
				}
			} else {
				cfg := GetConfig()
				lookback := 30
				if cfg != nil {
					lookback = cfg.GitIngest.LookbackDays
				}
				since = time.Now().AddDate(0, 0, -lookback)
			}

			app := GetLocalApp()
			if app == nil {
				return fmt.Errorf("local app service not available")
			}
			svc := app.Service()
			if svc == nil {
				return fmt.Errorf("local service not available")
			}

			cfg := GetConfig()
			maxFiles := 50
			if cfg != nil && cfg.GitIngest.MaxFilesPerCommit > 0 {
				maxFiles = cfg.GitIngest.MaxFilesPerCommit
			}

			mgr := git.NewLocalManager(svc)
			ing := git.NewIngester(mgr)
			opts := git.IngestOptions{
				Project:           project,
				RepoDir:           dir,
				Since:             since,
				DryRun:            dryRun,
				MaxFilesPerCommit: maxFiles,
			}

			fmt.Printf("Ingesting git history since %s for project %s...\n", since.Format(time.RFC3339), project)
			if err := ing.Ingest(context.Background(), opts); err != nil {
				return err
			}
			fmt.Println("Git ingest complete.")
			return nil
		},
	}

	cmd.Flags().StringVar(&sinceStr, "since", "", "Lookback duration (e.g., 7d, 30d) or RFC3339 date")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be ingested without persisting")
	return cmd
}

func resolveProjectName(dir string) string {
	if p := config.FindProjectConfig(dir); p != "" {
		// Derive project name from the directory containing .mairu.toml,
		// or we could parse the toml. For simplicity use the directory name.
		return filepath.Base(filepath.Dir(p))
	}
	return filepath.Base(dir)
}

func parseDurationOrDate(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		days, err := parseDays(s)
		if err == nil {
			return time.Now().AddDate(0, 0, -days), nil
		}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected duration like 7d or RFC3339 date")
}

func parseDays(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%dd", &n)
	return n, err
}
