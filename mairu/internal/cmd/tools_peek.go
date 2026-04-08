package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var peekLines string

func init() {
	peekCmd.Flags().StringVarP(&peekLines, "lines", "l", "", "Line range to extract (e.g., 50-100)")
	rootCmd.AddCommand(peekCmd)
}

type peekResult struct {
	F       string `json:"f"`
	Lines   string `json:"lines"`
	Content string `json:"content"`
}

var peekCmd = &cobra.Command{
	Use:   "peek <file>",
	Short: "AI-optimized exact line extraction (JSON)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		contentBytes, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if peekLines == "" {
			fmt.Fprintf(os.Stderr, "error: --lines (-l) flag is required for peek\n")
			os.Exit(1)
		}

		parts := strings.Split(peekLines, "-")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "error: invalid lines format, expected N-M\n")
			os.Exit(1)
		}

		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || start < 1 || end < start {
			fmt.Fprintf(os.Stderr, "error: invalid line range\n")
			os.Exit(1)
		}

		lines := strings.Split(string(contentBytes), "\n")

		if start > len(lines) {
			start = len(lines)
		}
		if end > len(lines) {
			end = len(lines)
		}

		// Adjust to 0-based index
		snippet := lines[start-1 : end]

		// Optional AI Optimization: strip common indentation to save tokens?
		// For now just return joined
		content := strings.Join(snippet, "\n")

		res := peekResult{
			F:       file,
			Lines:   peekLines,
			Content: content,
		}

		out, _ := json.Marshal(res)
		fmt.Println(string(out))
	},
}
