package cmd

import (
	"context"
	"fmt"
	"os"

	"mairu/internal/daemon"

	"github.com/spf13/cobra"
)

type daemonNoopManager struct{}

func (daemonNoopManager) UpsertFileContextNode(ctx context.Context, uri, name, abstractText, overviewText, content, parentURI, project string, metadata map[string]any) error {
	return nil
}
func (daemonNoopManager) DeleteContextNode(ctx context.Context, uri string) error { return nil }

func init() {
	daemonCmd := &cobra.Command{
		Use:   "daemon [dir]",
		Short: "Run local codebase daemon scan",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			if _, err := os.Stat(dir); err != nil {
				return err
			}
			d := daemon.New(daemonNoopManager{}, "default", dir, daemon.Options{})
			if err := d.ProcessAllFiles(context.Background()); err != nil {
				return err
			}
			fmt.Printf("Daemon scan complete for %s\n", dir)
			return nil
		},
	}
	rootCmd.AddCommand(daemonCmd)
}
