package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mairu/internal/ast"
	"mairu/internal/enricher"
)

const (
	CacheFilename             = ".contextfs-cache.json"
	cacheVersion              = 1
	defaultMaxFileSizeBytes   = 512 * 1024
	defaultProcessingDebounce = 200
	defaultConcurrency        = 8
	maxContentChars           = 16_000
)

var supportedExtensions = map[string]bool{
	".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true, ".py": true, ".go": true,
	".php": true, ".md": true, ".mdx": true, ".svelte": true,
}

var ignoredPathSegment = map[string]bool{
	"node_modules": true, "dist": true, "build": true,
}

// Manager defines the interface that the Daemon uses to persist context nodes
// created by analyzing files in the workspace.
type Manager interface {
	UpsertFileContextNode(ctx context.Context, uri, name, abstractText, overviewText, content, parentURI, project string, metadata map[string]any) error
	DeleteContextNode(ctx context.Context, uri string) error
}

// MarkdownSummarizer enriches markdown file summaries using an LLM.
// When non-nil, it replaces the heuristic abstract and overview with
// semantically richer descriptions optimized for retrieval.
type MarkdownSummarizer interface {
	SummarizeMarkdown(ctx context.Context, filename, content string) (abstract, overview string, err error)
}

// Options configures the background Daemon behavior for processing files.
type Options struct {
	MaxFileSizeBytes     int64
	ProcessingDebounceMs int
	Concurrency          int
	MarkdownSummarizer   MarkdownSummarizer
	EnricherPipeline     *enricher.Pipeline
}

type cacheEntry struct {
	Fingerprint string `json:"fingerprint"`
	ContentHash string `json:"contentHash"`
	PayloadHash string `json:"payloadHash"`
}

type cacheFile struct {
	Version int                   `json:"version"`
	Files   map[string]cacheEntry `json:"files"`
}

// Daemon monitors a directory for file changes and asynchronously processes
// them using AST parsing to extract and maintain natural language descriptions
// in the context graph database.
type Daemon struct {
	manager  Manager
	project  string
	watchDir string

	maxFileSizeBytes int64
	concurrency      int

	pendingFiles map[string]struct{}
	mu           sync.Mutex

	fileFingerprints  map[string]string
	fileContentHashes map[string]string
	nodePayloadHashes map[string]string

	describers       []ast.LanguageDescriber
	mdSummarizer     MarkdownSummarizer
	enricherPipeline *enricher.Pipeline
}

// New creates a new Daemon instance that watches a specified directory.
// It initializes the AST parsers, loads the local cache, and configures the processing pool.
func New(manager Manager, project, watchDir string, opts Options) *Daemon {
	maxSize := opts.MaxFileSizeBytes
	if maxSize <= 0 {
		maxSize = defaultMaxFileSizeBytes
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}
	return &Daemon{
		manager:           manager,
		project:           project,
		watchDir:          filepath.Clean(watchDir),
		maxFileSizeBytes:  maxSize,
		concurrency:       concurrency,
		pendingFiles:      map[string]struct{}{},
		fileFingerprints:  map[string]string{},
		fileContentHashes: map[string]string{},
		nodePayloadHashes: map[string]string{},
		describers: []ast.LanguageDescriber{
			ast.TypeScriptDescriber{},
			ast.TSXDescriber{},
			ast.VueDescriber{},
			ast.SvelteDescriber{},
			ast.GoDescriber{},
			ast.PythonDescriber{},
			ast.PHPDescriber{},
			ast.MarkdownDescriber{},
		},
		mdSummarizer:     opts.MarkdownSummarizer,
		enricherPipeline: opts.EnricherPipeline,
	}
}

func (d *Daemon) getDescriber(filePath string) ast.LanguageDescriber {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, desc := range d.describers {
		for _, e := range desc.Extensions() {
			if e == ext {
				return desc
			}
		}
	}
	return nil
}

func (d *Daemon) SaveCache() error {
	files := map[string]cacheEntry{}
	for abs, fp := range d.fileFingerprints {
		files[abs] = cacheEntry{
			Fingerprint: fp,
			ContentHash: d.fileContentHashes[abs],
			PayloadHash: d.nodePayloadHashes[abs],
		}
	}
	payload, err := json.MarshalIndent(cacheFile{Version: cacheVersion, Files: files}, "", "  ")
	if err != nil {
		return err
	}
	cachePath := filepath.Join(d.watchDir, CacheFilename)
	tmp := cachePath + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, cachePath)
}

func (d *Daemon) LoadCache() {
	cachePath := filepath.Join(d.watchDir, CacheFilename)
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		return
	}
	var c cacheFile
	if err := json.Unmarshal(raw, &c); err != nil || c.Version != cacheVersion {
		return
	}
	for abs, entry := range c.Files {
		d.fileFingerprints[abs] = entry.Fingerprint
		d.fileContentHashes[abs] = entry.ContentHash
		d.nodePayloadHashes[abs] = entry.PayloadHash
	}
}

func (d *Daemon) QueueFile(filePath string) {
	if !d.shouldProcessFile(filePath) {
		return
	}
	d.mu.Lock()
	d.pendingFiles[filepath.Clean(filePath)] = struct{}{}
	d.mu.Unlock()
}

func (d *Daemon) ProcessPendingFiles(ctx context.Context) error {
	d.mu.Lock()
	files := make([]string, 0, len(d.pendingFiles))
	for f := range d.pendingFiles {
		files = append(files, f)
	}
	d.pendingFiles = map[string]struct{}{}
	d.mu.Unlock()
	if err := d.runWithConcurrency(ctx, files, d.concurrency, d.ProcessFile); err != nil {
		return err
	}
	return d.SaveCache()
}

func (d *Daemon) ProcessAllFiles(ctx context.Context) error {
	files := d.discoverSourceFiles(d.watchDir)
	if err := d.runWithConcurrency(ctx, files, d.concurrency, d.ProcessFile); err != nil {
		return err
	}
	return d.SaveCache()
}

func (d *Daemon) HandleFileDelete(ctx context.Context, filePath string) error {
	abs := filepath.Clean(filePath)
	d.mu.Lock()
	delete(d.pendingFiles, abs)
	delete(d.fileFingerprints, abs)
	delete(d.fileContentHashes, abs)
	delete(d.nodePayloadHashes, abs)
	d.mu.Unlock()
	uri, err := d.fileToURI(abs)
	if err != nil {
		return err
	}
	return d.manager.DeleteContextNode(ctx, uri)
}

func (d *Daemon) ProcessFile(ctx context.Context, filePath string) error {
	abs := filepath.Clean(filePath)
	if !d.shouldProcessFile(abs) {
		return nil
	}
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return nil
	}
	if st.Size() > d.maxFileSizeBytes {
		return nil
	}
	fp := fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixMilli())

	d.mu.Lock()
	existingFp := d.fileFingerprints[abs]
	d.mu.Unlock()

	if existingFp == fp {
		return nil
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	contentHash := hashText(string(raw))

	d.mu.Lock()
	existingContentHash := d.fileContentHashes[abs]
	d.mu.Unlock()

	if existingContentHash == contentHash {
		d.mu.Lock()
		d.fileFingerprints[abs] = fp
		d.mu.Unlock()
		return nil
	}

	summary, err := d.summarizeSourceFile(ctx, abs, string(raw))
	if err != nil {
		return err
	}
	metadata := map[string]any{
		"type":        "file",
		"path":        abs,
		"source_hash": contentHash,
		"logic_graph": summary.LogicGraph,
	}

	// Run enricher pipeline if configured
	if d.enricherPipeline != nil {
		rel, err := filepath.Rel(d.watchDir, abs)
		if err != nil {
			return err
		}
		fc := &enricher.FileContext{
			FilePath: abs,
			RelPath:  filepath.ToSlash(rel),
			WatchDir: d.watchDir,
			Metadata: metadata,
		}
		d.enricherPipeline.Run(ctx, fc)
	}

	payloadHash := hashText(summary.Abstract + "\n" + summary.Overview + "\n" + summary.Content + "\n" + mustJSON(metadata))

	d.mu.Lock()
	existingPayloadHash := d.nodePayloadHashes[abs]
	d.mu.Unlock()

	if existingPayloadHash == payloadHash {
		d.mu.Lock()
		d.fileFingerprints[abs] = fp
		d.fileContentHashes[abs] = contentHash
		d.mu.Unlock()
		return nil
	}

	uri, err := d.fileToURI(abs)
	if err != nil {
		return err
	}
	parentURI, err := d.fileToParentURI(abs)
	if err != nil {
		return err
	}
	if err := d.manager.UpsertFileContextNode(
		ctx,
		uri,
		filepath.Base(abs),
		summary.Abstract,
		summary.Overview,
		summary.Content,
		parentURI,
		d.project,
		metadata,
	); err != nil {
		return err
	}

	d.mu.Lock()
	d.fileFingerprints[abs] = fp
	d.fileContentHashes[abs] = contentHash
	d.nodePayloadHashes[abs] = payloadHash
	d.mu.Unlock()

	fmt.Printf("[Daemon] Updated AST context file=%s\n", filepath.Base(abs))
	return nil
}

func (d *Daemon) discoverSourceFiles(dir string) []string {
	var out []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if ignoredPathSegment[e.Name()] || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			out = append(out, d.discoverSourceFiles(full)...)
			continue
		}
		if d.shouldProcessFile(full) {
			out = append(out, full)
		}
	}
	return out
}

func (d *Daemon) shouldProcessFile(filePath string) bool {
	abs := filepath.Clean(filePath)
	rel, err := filepath.Rel(d.watchDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false
	}
	for _, s := range strings.Split(filepath.ToSlash(rel), "/") {
		if ignoredPathSegment[s] || strings.HasPrefix(s, ".") {
			return false
		}
	}
	return supportedExtensions[strings.ToLower(filepath.Ext(abs))]
}

func (d *Daemon) runWithConcurrency(ctx context.Context, items []string, concurrency int, fn func(context.Context, string) error) error {
	if len(items) == 0 {
		return nil
	}
	if concurrency < 1 {
		concurrency = 1
	}
	ch := make(chan string)
	errCh := make(chan error, len(items))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for item := range ch {
			if err := fn(ctx, item); err != nil {
				errCh <- err
			}
		}
	}
	n := concurrency
	if n > len(items) {
		n = len(items)
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go worker()
	}

	go func() {
		for _, item := range items {
			ch <- item
		}
		close(ch)
	}()

	wg.Wait()
	close(errCh)

	var firstErr error
	for err := range errCh {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (d *Daemon) fileToURI(filePath string) (string, error) {
	rel, err := filepath.Rel(d.watchDir, filePath)
	if err != nil {
		return "", err
	}
	return "contextfs://" + d.project + "/" + filepath.ToSlash(rel), nil
}

func (d *Daemon) fileToParentURI(filePath string) (string, error) {
	rel, err := filepath.Rel(d.watchDir, filePath)
	if err != nil {
		return "", err
	}
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." || dir == "" {
		return "contextfs://" + d.project, nil
	}
	return "contextfs://" + d.project + "/" + dir, nil
}
