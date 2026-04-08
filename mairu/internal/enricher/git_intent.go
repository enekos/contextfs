package enricher

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var conventionalPrefixRe = regexp.MustCompile(`^(\w+)(?:\(.+?\))?:\s+(.+)`)

// GitIntentEnricher annotates files with why they exist, extracted from git commit
// messages. It parses conventional commit prefixes and produces a human-readable
// intent summary. No LLM calls — heuristic only (LLM batch summarization is a
// future enhancement).
type GitIntentEnricher struct {
	MaxCommits int // how many recent commits to analyze; 0 defaults to 20
}

func (e *GitIntentEnricher) Name() string { return "git_intent" }

func (e *GitIntentEnricher) Enrich(ctx context.Context, fc *FileContext) error {
	maxCommits := e.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 20
	}

	messages, err := gitCommitMessages(ctx, fc.WatchDir, fc.RelPath, maxCommits)
	if err != nil || len(messages) == 0 {
		return nil
	}

	prefixCounts := map[string]int{}
	var subjects []string

	for _, msg := range messages {
		match := conventionalPrefixRe.FindStringSubmatch(msg)
		if match != nil {
			prefix := strings.ToLower(match[1])
			prefixCounts[prefix]++
			subjects = append(subjects, match[2])
		} else {
			subjects = append(subjects, msg)
		}
	}

	intent := buildIntentSummary(subjects, prefixCounts, len(messages))

	fc.Metadata["enrichment_intent"] = intent
	if len(prefixCounts) > 0 {
		fc.Metadata["enrichment_commit_prefixes"] = prefixCounts
	}

	return nil
}

func buildIntentSummary(subjects []string, prefixes map[string]int, totalCommits int) string {
	var parts []string

	// Summarize commit type distribution
	if len(prefixes) > 0 {
		var typeParts []string
		for prefix, count := range prefixes {
			typeParts = append(typeParts, fmt.Sprintf("%d %s", count, prefix))
		}
		parts = append(parts, fmt.Sprintf("Commit history (%d commits): %s.", totalCommits, strings.Join(typeParts, ", ")))
	} else {
		parts = append(parts, fmt.Sprintf("Commit history: %d commits (no conventional prefixes).", totalCommits))
	}

	// Include the most recent commit subjects (up to 5) for context
	maxSubjects := 5
	if len(subjects) < maxSubjects {
		maxSubjects = len(subjects)
	}
	if maxSubjects > 0 {
		recent := subjects[:maxSubjects]
		parts = append(parts, "Recent changes: "+strings.Join(recent, "; ")+".")
	}

	return strings.Join(parts, " ")
}

// gitCommitMessages returns the subject line of the most recent N commits touching relPath.
func gitCommitMessages(ctx context.Context, repoDir, relPath string, maxCommits int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		fmt.Sprintf("-%d", maxCommits),
		"--format=%s",
		"--", relPath,
	)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var messages []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			messages = append(messages, line)
		}
	}
	return messages, nil
}
