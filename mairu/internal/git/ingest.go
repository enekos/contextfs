package git

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mairu/internal/ast"
	"mairu/internal/contextsrv"
)

// Manager is the interface the git ingester uses to persist context nodes.
type Manager interface {
	UpsertFileContextNode(ctx context.Context, uri, name, abstractText, overviewText, content, parentURI, project string, metadata map[string]any, createdAt *time.Time) error
	GetContextNode(ctx context.Context, uri string) (contextsrv.ContextNode, error)
}

type IngestOptions struct {
	Project           string
	RepoDir           string
	Since             time.Time
	DryRun            bool
	MaxFilesPerCommit int
	MaxContentChars   int
}

type Ingester struct {
	Manager Manager
}

func NewIngester(mgr Manager) *Ingester {
	return &Ingester{Manager: mgr}
}

func (i *Ingester) Ingest(ctx context.Context, opts IngestOptions) error {
	if opts.MaxFilesPerCommit <= 0 {
		opts.MaxFilesPerCommit = 50
	}
	if opts.MaxContentChars <= 0 {
		opts.MaxContentChars = 16000
	}

	commits, err := listCommits(opts.RepoDir, opts.Since)
	if err != nil {
		return fmt.Errorf("list commits: %w", err)
	}
	if len(commits) == 0 {
		fmt.Println("No commits found in the specified range.")
		return nil
	}

	for _, c := range commits {
		if err := i.ingestCommit(ctx, opts, c); err != nil {
			return fmt.Errorf("ingest commit %s: %w", c.Hash, err)
		}
	}

	return nil
}

type commitInfo struct {
	Hash    string
	Date    time.Time
	Author  string
	Subject string
	Body    string
}

func listCommits(repoDir string, since time.Time) ([]commitInfo, error) {
	cmd := exec.Command("git", "log",
		fmt.Sprintf("--since=%s", since.Format(time.RFC3339)),
		"--format=%H%x00%aI%x00%an%x00%s%x00%b%x00%x01",
		"--reverse",
	)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []commitInfo
	entries := strings.Split(string(out), "\x01")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "\x00", 5)
		if len(parts) < 4 {
			continue
		}
		date, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			continue
		}
		body := ""
		if len(parts) > 4 {
			body = strings.TrimSpace(parts[4])
		}
		commits = append(commits, commitInfo{
			Hash:    parts[0],
			Date:    date,
			Author:  parts[2],
			Subject: parts[3],
			Body:    body,
		})
	}
	return commits, nil
}

func (i *Ingester) ingestCommit(ctx context.Context, opts IngestOptions, c commitInfo) error {
	files, err := changedFiles(opts.RepoDir, c.Hash)
	if err != nil {
		return fmt.Errorf("changed files: %w", err)
	}

	commitURI := fmt.Sprintf("contextfs://%s/git/commit/%s", opts.Project, c.Hash)
	shortHash := c.Hash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	fullMessage := c.Subject
	if c.Body != "" {
		fullMessage = c.Subject + "\n\n" + c.Body
	}

	commitContent := fmt.Sprintf("Commit: %s\nAuthor: %s\nDate: %s\n\n%s\n\nFiles:\n%s",
		c.Hash, c.Author, c.Date.Format(time.RFC3339), fullMessage, strings.Join(files, "\n"))
	if len(commitContent) > opts.MaxContentChars {
		commitContent = commitContent[:opts.MaxContentChars] + "\n...TRUNCATED"
	}

	if !opts.DryRun {
		meta := map[string]any{
			"type":   "git_commit",
			"hash":   c.Hash,
			"author": c.Author,
			"date":   c.Date.Format(time.RFC3339),
			"files":  files,
		}
		if err := i.Manager.UpsertFileContextNode(ctx, commitURI, shortHash, c.Subject, commitContent, commitContent, "", opts.Project, meta, &c.Date); err != nil {
			return fmt.Errorf("upsert commit node: %w", err)
		}
	}

	processed := 0
	for _, f := range files {
		if processed >= opts.MaxFilesPerCommit {
			break
		}
		if !isSupportedFile(f) {
			continue
		}
		if err := i.ingestFile(ctx, opts, c, f, commitURI, shortHash); err != nil {
			// Log and continue; don't fail the whole commit for one file
			fmt.Printf("[git ingest] warning: skipping %s@%s: %v\n", f, shortHash, err)
			continue
		}
		processed++
	}

	return nil
}

func changedFiles(repoDir, hash string) ([]string, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "--root", hash)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}
	return files, scanner.Err()
}

func (i *Ingester) ingestFile(ctx context.Context, opts IngestOptions, c commitInfo, file, commitURI, shortHash string) error {
	snapshotURI := fmt.Sprintf("contextfs://%s/git/snapshot/%s/%s", opts.Project, c.Hash, file)
	liveURI := fmt.Sprintf("contextfs://%s/%s", opts.Project, file)

	// Check if snapshot already exists to skip expensive AST parsing
	existing, err := i.Manager.GetContextNode(ctx, snapshotURI)
	if err == nil && existing.URI != "" {
		// Re-use existing snapshot; only ensure diff node exists
		return i.upsertDiffNode(ctx, opts, c, file, commitURI, snapshotURI, liveURI, shortHash, "")
	}

	src, err := showFileAtCommit(opts.RepoDir, c.Hash, file)
	if err != nil {
		return fmt.Errorf("show file: %w", err)
	}
	if isBinaryContent(src) {
		return fmt.Errorf("binary file")
	}

	describer := getDescriber(file)
	var summary ast.FileSummary
	if describer != nil {
		fg := describer.ExtractFileGraph(file, src)
		abstract := fg.FileSummary
		summary = ast.SummarizeFile(file, describer.LanguageID(), abstract, fg, opts.MaxContentChars)
	} else {
		abstract := "File snapshot"
		if len(src) == 0 {
			abstract = "Empty file snapshot"
		}
		summary = ast.FileSummary{
			Abstract: abstract,
			Overview: abstract,
			Content:  src,
		}
		if len(summary.Content) > opts.MaxContentChars {
			summary.Content = summary.Content[:opts.MaxContentChars] + "\n\n...TRUNCATED"
		}
	}

	if !opts.DryRun {
		snapMeta := map[string]any{
			"type":     "git_snapshot",
			"hash":     c.Hash,
			"file":     file,
			"live_uri": liveURI,
		}
		if err := i.Manager.UpsertFileContextNode(ctx, snapshotURI, filepath.Base(file), summary.Abstract, summary.Overview, summary.Content, commitURI, opts.Project, snapMeta, &c.Date); err != nil {
			return fmt.Errorf("upsert snapshot node: %w", err)
		}
	}

	return i.upsertDiffNode(ctx, opts, c, file, commitURI, snapshotURI, liveURI, shortHash, summary.Content)
}

func (i *Ingester) upsertDiffNode(ctx context.Context, opts IngestOptions, c commitInfo, file, commitURI, snapshotURI, liveURI, shortHash, astSummary string) error {
	if opts.DryRun {
		return nil
	}

	diffText, err := getDiff(opts.RepoDir, c.Hash, file)
	if err != nil {
		// Diff failure is non-fatal; we can still create a diff node without the patch
		diffText = ""
	}

	abstract := fmt.Sprintf("Changed %s: %s", file, c.Subject)
	overview := diffText
	if len(overview) > opts.MaxContentChars {
		overview = overview[:opts.MaxContentChars] + "\n...TRUNCATED"
	}

	content := diffText
	if astSummary != "" {
		content = diffText + "\n\n--- AST Summary ---\n\n" + astSummary
	}
	if len(content) > opts.MaxContentChars {
		content = content[:opts.MaxContentChars] + "\n\n...TRUNCATED"
	}

	additions, deletions := countDiffStats(diffText)
	meta := map[string]any{
		"type":         "git_diff",
		"hash":         c.Hash,
		"file":         file,
		"snapshot_uri": snapshotURI,
		"live_uri":     liveURI,
		"additions":    additions,
		"deletions":    deletions,
	}

	diffURI := fmt.Sprintf("contextfs://%s/git/diff/%s/%s", opts.Project, c.Hash, file)
	return i.Manager.UpsertFileContextNode(ctx, diffURI, filepath.Base(file), abstract, overview, content, commitURI, opts.Project, meta, &c.Date)
}

func showFileAtCommit(repoDir, hash, file string) (string, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", hash, file))
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func getDiff(repoDir, hash, file string) (string, error) {
	// Try with parent first
	parent := hash + "^"
	cmd := exec.Command("git", "diff", parent, hash, "--", file)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		// May be root commit; fallback to show entire commit for this file
		cmd = exec.Command("git", "show", hash, "--", file)
		cmd.Dir = repoDir
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}
	return string(out), nil
}

func isSupportedFile(file string) bool {
	ext := strings.ToLower(filepath.Ext(file))
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".py", ".go", ".php", ".md", ".mdx", ".svelte":
		return true
	default:
		return false
	}
}

func isBinaryContent(s string) bool {
	for i := 0; i < len(s) && i < 8000; i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}

func getDescriber(filePath string) ast.LanguageDescriber {
	ext := strings.ToLower(filepath.Ext(filePath))
	describers := []ast.LanguageDescriber{
		ast.TypeScriptDescriber{},
		ast.TSXDescriber{},
		ast.VueDescriber{},
		ast.SvelteDescriber{},
		ast.GoDescriber{},
		ast.PythonDescriber{},
		ast.PHPDescriber{},
		ast.MarkdownDescriber{},
	}
	for _, desc := range describers {
		for _, e := range desc.Extensions() {
			if e == ext {
				return desc
			}
		}
	}
	return nil
}

func countDiffStats(diff string) (additions, deletions int) {
	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "@@") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			additions++
		} else if strings.HasPrefix(line, "-") {
			deletions++
		}
	}
	return
}

// localManager adapts a contextsrv.NodeService to the git.Manager interface.
type localManager struct {
	svc contextsrv.NodeService
}

func NewLocalManager(svc contextsrv.NodeService) Manager {
	return &localManager{svc: svc}
}

func (m *localManager) UpsertFileContextNode(ctx context.Context, uri, name, abstractText, overviewText, content, parentURI, project string, metadata map[string]any, createdAt *time.Time) error {
	var p *string
	if parentURI != "" {
		p = &parentURI
	}
	metaJSON, _ := json.Marshal(metadata)
	_, err := m.svc.CreateContextNode(contextsrv.ContextCreateInput{
		URI:       uri,
		Project:   project,
		ParentURI: p,
		Name:      name,
		Abstract:  abstractText,
		Overview:  overviewText,
		Content:   content,
		Metadata:  metaJSON,
		CreatedAt: createdAt,
	})
	return err
}

func (m *localManager) GetContextNode(ctx context.Context, uri string) (contextsrv.ContextNode, error) {
	return m.svc.GetContextNode(uri)
}
