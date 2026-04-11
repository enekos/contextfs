package cmd

import (
	"fmt"
	"log/slog"
	"mairu/internal/agent"
	"mairu/internal/prompts"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var (
	minionMaxRetries int
	minionCouncil    bool
)

func NewMinionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "minion [prompt]",
		Short: "Run Mairu in unattended, one-shot Minion Mode",
		Long: `Minion Mode executes tasks completely unattended. It will automatically approve shell commands, 
run verification checks, attempt to fix issues (up to --max-retries), and open a Pull Request.
Ideal for executing from background jobs or automation pipelines.`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var prompt string
			if len(args) > 0 {
				prompt = strings.Join(args, " ")
			}

			if minionGithubIssue != "" {
				issueData, err := fetchGitHubContext("issue", minionGithubIssue)
				if err != nil {
					slog.Error("Failed to fetch github issue", "error", err)
					os.Exit(1)
				}
				prompt = fmt.Sprintf("Resolve the following GitHub Issue:\n\n%s\n\nAdditional user prompt: %s", issueData, prompt)
			} else if minionGithubPR != "" {
				prData, err := fetchGitHubContext("pr", minionGithubPR)
				if err != nil {
					slog.Error("Failed to fetch github PR", "error", err)
					os.Exit(1)
				}
				prompt = fmt.Sprintf("Address the following PR feedback:\n\n%s\n\nAdditional user prompt: %s", prData, prompt)
			} else if prompt == "" {
				slog.Error("Either a prompt, --github-issue, or --github-pr is required")
				cmd.Usage()
				os.Exit(1)
			}

			runMinion(prompt)
		},
	}
	cmd.Flags().IntVar(&minionMaxRetries, "max-retries", 2, "Maximum attempts to fix failing tests/linters")
	cmd.Flags().StringVar(&minionGithubIssue, "github-issue", "", "GitHub Issue number to resolve")
	cmd.Flags().StringVar(&minionGithubPR, "github-pr", "", "GitHub PR number to review and fix")
	cmd.Flags().BoolVar(&minionCouncil, "council", false, "Enable council mode with expert reviewers before execution")
	return cmd
}

var (
	minionGithubIssue string
	minionGithubPR    string
)

func fetchGitHubContext(entityType, number string) (string, error) {
	var cmd *exec.Cmd
	var out []byte
	var err error

	if entityType == "issue" {
		cmd = exec.Command("gh", "issue", "view", number, "--comments")
		out, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("gh issue view failed: %s, output: %s", err, string(out))
		}
		return string(out), nil
	} else {
		cmd = exec.Command("gh", "pr", "view", number, "--comments")
		viewOut, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("gh pr view failed: %s, output: %s", err, string(viewOut))
		}

		diffCmd := exec.Command("gh", "pr", "diff", number)
		diffOut, diffErr := diffCmd.CombinedOutput()
		if diffErr != nil {
			// Don't fail the whole command if diff fails, just return what we have
			slog.Warn("Failed to fetch PR diff", "error", diffErr)
			return string(viewOut), nil
		}

		return fmt.Sprintf("%s\n\n=== PR DIFF ===\n%s", string(viewOut), string(diffOut)), nil
	}
}

func runMinion(userPrompt string) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		slog.Error("Gemini API key not found. Please run 'mairu setup' or set GEMINI_API_KEY environment variable.")
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	cfg := GetAgentConfig()
	cfg.Unattended = true
	cfg.Council.Enabled = minionCouncil
	a, err := agent.New(cwd, apiKey, cfg)
	if err != nil {
		slog.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}
	defer a.Close()

	// Minion specific instructions wrapping the user prompt
	minionPrompt := prompts.Render("minion_prompt", struct {
		Task       string
		MaxRetries int
	}{
		Task:       userPrompt,
		MaxRetries: minionMaxRetries,
	})

	var hasError bool
	maxAttempts := minionMaxRetries + 1
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		hasError = runMinionAttempt(a, minionPrompt, attempt, maxAttempts)
		if !hasError {
			break
		}
		if attempt < maxAttempts {
			fmt.Printf("\n⚠️ Attempt %d/%d failed. Retrying...\n", attempt, maxAttempts)
		}
	}

	if hasError {
		os.Exit(1)
	}

	if !minionCouncil {
		return
	}

	discoveredPR := discoverCurrentPRNumber()
	reviewPR := resolvePRReviewTarget(minionGithubPR, discoveredPR)
	if reviewPR == "" {
		return
	}

	fmt.Printf("\n🔎 Launching PR reviewer council for PR #%s\n", reviewPR)
	reviewOutput, err := runPRReviewerCouncil(cwd, apiKey, reviewPR)
	if err != nil {
		fmt.Printf("⚠️ PR reviewer council failed: %v\n", err)
		return
	}
	fmt.Println("\n📌 PR reviewer council suggestions:")
	fmt.Println(reviewOutput)
}

func runMinionAttempt(a *agent.Agent, prompt string, attempt, maxAttempts int) bool {
	if maxAttempts > 1 {
		fmt.Printf("\n🚀 Minion attempt %d/%d\n\n", attempt, maxAttempts)
	}

	outChan := make(chan agent.AgentEvent)
	go a.RunStream(prompt, outChan)

	var hasError bool
	for ev := range outChan {
		switch ev.Type {
		case "status":
			fmt.Printf("ℹ️  %s\n", ev.Content)
		case "tool_call":
			fmt.Printf("🔧 Executing: %s\n", ev.ToolName)
		case "tool_result":
			if verbose {
				fmt.Printf("✅ Tool %s finished\n", ev.ToolName)
			}
		case "text":
			fmt.Print(ev.Content)
		case "diff":
			fmt.Printf("\n%s\n\n", ev.Content)
		case "error":
			fmt.Printf("\n❌ Error: %s\n", ev.Content)
			hasError = true
		case "done":
			fmt.Println("\n🏁 Minion finished.")
		}
	}
	return hasError
}

type prReviewerRole struct {
	Name  string
	Focus string
}

type prReviewerResult struct {
	role    string
	content string
	err     error
}

type reviewerStatus string

const (
	reviewerStatusOK     reviewerStatus = "ok"
	reviewerStatusFailed reviewerStatus = "failed"
	reviewerStatusEmpty  reviewerStatus = "empty"
)

type prReviewerOutcome struct {
	role    string
	status  reviewerStatus
	content string
	err     string
}

func runPRReviewerCouncil(cwd, apiKey, prNumber string) (string, error) {
	prData, err := fetchGitHubContext("pr", prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PR context: %w", err)
	}

	roles := []prReviewerRole{
		{
			Name:  "App Developer",
			Focus: "Validate feature behavior and practical product heuristics against surrounding code paths.",
		},
		{
			Name:  "Developer Evangelist",
			Focus: "Review style, architecture consistency, pitfalls, and performance concerns.",
		},
		{
			Name:  "Tests Evangelist",
			Focus: "Check that relevant tests are added or updated, and identify testing gaps.",
		},
	}

	resultsCh := make(chan prReviewerResult, len(roles))
	var wg sync.WaitGroup
	for _, role := range roles {
		wg.Add(1)
		go func(r prReviewerRole) {
			defer wg.Done()
			out, reviewErr := runSinglePRReviewer(cwd, apiKey, prNumber, prData, r)
			resultsCh <- prReviewerResult{
				role:    r.Name,
				content: out,
				err:     reviewErr,
			}
		}(role)
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	outcomes := map[string]prReviewerOutcome{}
	successCount := 0
	for res := range resultsCh {
		if res.err != nil {
			outcomes[res.role] = prReviewerOutcome{
				role:   res.role,
				status: reviewerStatusFailed,
				err:    res.err.Error(),
			}
			continue
		}
		content := strings.TrimSpace(res.content)
		if content == "" {
			outcomes[res.role] = prReviewerOutcome{
				role:   res.role,
				status: reviewerStatusEmpty,
			}
			continue
		}
		successCount++
		outcomes[res.role] = prReviewerOutcome{
			role:    res.role,
			status:  reviewerStatusOK,
			content: content,
		}
	}

	if successCount == 0 {
		return "", fmt.Errorf("all PR council reviewers failed or returned empty output")
	}
	return runProductLeadSynthesis(cwd, apiKey, prNumber, outcomes)
}

func runSinglePRReviewer(cwd, apiKey, prNumber, prContext string, role prReviewerRole) (string, error) {
	cfg := GetAgentConfig()
	cfg.Unattended = true
	cfg.Council.Enabled = false

	a, err := agent.New(cwd, apiKey, cfg)
	if err != nil {
		return "", err
	}
	defer a.Close()

	prompt := prompts.Render("council_pr_reviewer_expert", struct {
		PRNumber string
		PRData   string
		Role     string
		Focus    string
	}{
		PRNumber: prNumber,
		PRData:   prContext,
		Role:     role.Name,
		Focus:    role.Focus,
	})

	outChan := make(chan agent.AgentEvent, 100)
	go a.RunStream(prompt, outChan)

	var b strings.Builder
	var eventErr error
	for ev := range outChan {
		if ev.Type == "text" {
			b.WriteString(ev.Content)
		}
		if ev.Type == "error" {
			eventErr = fmt.Errorf("%s", ev.Content)
		}
	}
	if eventErr != nil {
		return "", eventErr
	}
	return strings.TrimSpace(b.String()), nil
}

func runProductLeadSynthesis(cwd, apiKey, prNumber string, findings map[string]prReviewerOutcome) (string, error) {
	cfg := GetAgentConfig()
	cfg.Unattended = true
	cfg.Council.Enabled = false

	a, err := agent.New(cwd, apiKey, cfg)
	if err != nil {
		return "", err
	}
	defer a.Close()

	prompt := prompts.Render("council_pr_reviewer_product_lead", struct {
		PRNumber string
		Findings string
	}{
		PRNumber: prNumber,
		Findings: formatReviewerFindings(findings),
	})

	outChan := make(chan agent.AgentEvent, 100)
	go a.RunStream(prompt, outChan)

	var b strings.Builder
	var eventErr error
	for ev := range outChan {
		if ev.Type == "text" {
			b.WriteString(ev.Content)
		}
		if ev.Type == "error" {
			eventErr = fmt.Errorf("%s", ev.Content)
		}
	}
	if eventErr != nil {
		return "", eventErr
	}
	return strings.TrimSpace(b.String()), nil
}

func discoverCurrentPRNumber() string {
	cmd := exec.Command("gh", "pr", "view", "--json", "number", "--jq", ".number")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func resolvePRReviewTarget(explicitPR, discoveredPR string) string {
	if strings.TrimSpace(explicitPR) != "" {
		return strings.TrimSpace(explicitPR)
	}
	return strings.TrimSpace(discoveredPR)
}

func formatReviewerFindings(findings map[string]prReviewerOutcome) string {
	roles := make([]string, 0, len(findings))
	for role := range findings {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	var b strings.Builder
	for _, role := range roles {
		outcome := findings[role]
		b.WriteString("## ")
		b.WriteString(role)
		b.WriteString("\n")
		b.WriteString("status: ")
		b.WriteString(string(outcome.status))
		b.WriteString("\n")
		switch outcome.status {
		case reviewerStatusOK:
			b.WriteString(strings.TrimSpace(outcome.content))
		case reviewerStatusFailed:
			b.WriteString("failure_reason: ")
			b.WriteString(strings.TrimSpace(outcome.err))
		case reviewerStatusEmpty:
			b.WriteString("reviewer returned no findings")
		}
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}
