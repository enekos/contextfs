package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

var scanBudget int
var scanContext int

func init() {
	scanCmd.Flags().IntVar(&scanBudget, "budget", 2000, "Token budget circuit breaker")
	scanCmd.Flags().IntVarP(&scanContext, "context", "C", 0, "Number of context lines around match")
	rootCmd.AddCommand(scanCmd)
}

type scanMatch struct {
	F string `json:"f"`
	L int    `json:"l"`
	C string `json:"c"`
}

type scanResult struct {
	BudgetHit bool        `json:"budget_hit"`
	Matches   []scanMatch `json:"matches"`
}

var scanCmd = &cobra.Command{
	Use:   "scan <regex> [dir]",
	Short: "AI-optimized semantic search with token budget (JSON)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pattern := args[0]
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error compiling regex: %v\n", err)
			os.Exit(1)
		}

		var ignorer *ignore.GitIgnore
		if gi, err := ignore.CompileIgnoreFile(filepath.Join(dir, ".gitignore")); err == nil {
			ignorer = gi
		}

		res := scanResult{Matches: []scanMatch{}}
		var currentBytes int
		// roughly 4 bytes per token
		maxBytes := scanBudget * 4

		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if path == dir {
				return nil
			}

			rel, _ := filepath.Rel(dir, path)
			if rel == ".git" {
				return filepath.SkipDir
			}
			if ignorer != nil && ignorer.MatchesPath(rel) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".png" || ext == ".jpg" || ext == ".exe" || ext == ".bin" {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					if re.MatchString(line) {
						startIdx := i - scanContext
						if startIdx < 0 {
							startIdx = 0
						}
						endIdx := i + scanContext
						if endIdx >= len(lines) {
							endIdx = len(lines) - 1
						}

						var snippet []string
						for j := startIdx; j <= endIdx; j++ {
							snippet = append(snippet, strings.TrimSpace(lines[j]))
						}

						joined := strings.Join(snippet, "\n")
						matchBytes := len(rel) + len(joined) + 20

						if currentBytes+matchBytes > maxBytes {
							res.BudgetHit = true
							return filepath.SkipDir
						}
						currentBytes += matchBytes
						res.Matches = append(res.Matches, scanMatch{
							F: rel,
							L: i + 1,
							C: joined,
						})
					}
				}
			}
			return nil
		})

		out, _ := json.Marshal(res)
		fmt.Println(string(out))
	},
}
