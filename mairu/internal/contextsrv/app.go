package contextsrv

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mairu/internal/config"
	"mairu/internal/llm"
)

// Config defines configuration parameters for starting the Context Server.
// It includes server settings, database connections, API keys, and projector configuration.
type Config struct {
	Port              int
	SQLiteDSN         string
	MeiliURL          string
	MeiliAPIKey       string
	GeminiAPIKey      string
	KimiAPIKey        string
	LLMProvider       string // "gemini" or "kimi"
	LLMModel          string
	LLMBaseURL        string
	AuthToken         string
	EnableProjector   bool
	ProjectorEvery    time.Duration
	ProjectorBatch    int
	ReadTimeout       time.Duration
	ModerationEnabled bool
	EmbeddingProvider string
	EmbeddingBaseURL  string
	EmbeddingAPIKey   string
	EmbeddingModel    string
}

// App represents the Context Server application instance.
// It bundles the configuration, repository, projector, and HTTP server to manage the application lifecycle.
type App struct {
	cfg       Config
	repo      *SQLiteRepository
	projector *Projector
	server    *http.Server
	svc       Service
}

// Service returns the underlying service for external consumers.
func (a *App) Service() Service {
	return a.svc
}

// Repo returns the underlying SQLiteRepository.
func (a *App) Repo() *SQLiteRepository {
	return a.repo
}

// NewApp initializes and returns a new App instance using the provided Config.
// It sets up the repository, LLM provider, indexing service, handler, and an HTTP server.
func NewApp(cfg Config) (*App, error) {
	var repo *SQLiteRepository
	var err error
	if cfg.SQLiteDSN != "" {
		repo, err = NewSQLiteRepository(cfg.SQLiteDSN)
		if err != nil {
			return nil, err
		}
	}

	// Initialize LLM provider
	var llmProvider llm.Provider
	providerCfg := llm.ProviderConfig{
		Type:    llm.ProviderType(cfg.LLMProvider),
		APIKey:  cfg.GeminiAPIKey,
		Model:   cfg.LLMModel,
		BaseURL: cfg.LLMBaseURL,
	}
	if providerCfg.Type == "" {
		providerCfg.Type = llm.ProviderGemini
	}
	if providerCfg.Type == llm.ProviderKimi {
		providerCfg.APIKey = cfg.KimiAPIKey
	}

	if providerCfg.APIKey != "" {
		client, err := llm.NewProvider(providerCfg)
		if err != nil {
			slog.Warn("failed to initialize LLM provider", "provider", providerCfg.Type, "error", err)
		} else {
			llmProvider = client
		}
	}

	embedder, err := llm.NewEmbedder(config.EmbeddingConfig{
		Provider: cfg.EmbeddingProvider,
		Model:    cfg.EmbeddingModel,
		BaseURL:  cfg.EmbeddingBaseURL,
		APIKey:   cfg.EmbeddingAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	indexer := NewMeiliIndexer(cfg.MeiliURL, cfg.MeiliAPIKey, embedder)
	ensureCtx, ensureCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ensureCancel()
	done := make(chan error, 1)
	go func() {
		done <- indexer.EnsureIndexes()
	}()
	select {
	case <-ensureCtx.Done():
		slog.Warn("Meilisearch ensure indexes timed out, continuing without search index")
	case err := <-done:
		if err != nil {
			slog.Warn("failed to ensure meilisearch indexes", "error", err)
		}
	}

	var llmClient LLMClient
	if llmProvider != nil {
		llmClient = llmProvider
	}

	var repoIntf Repository
	if repo != nil {
		repoIntf = repo
	}

	svc := NewServiceWithSearch(repoIntf, indexer, llmClient, embedder, cfg.ModerationEnabled)
	handler := NewHandler(svc, cfg.AuthToken)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadTimeout,
	}

	var projector *Projector
	if repo != nil {
		projector = NewProjector(repo, indexer, embedder)
	}

	return &App{
		cfg:       cfg,
		repo:      repo,
		projector: projector,
		server:    srv,
		svc:       svc,
	}, nil
}

// Start begins the execution of the application.
// It starts the background projector (if enabled) and listens on the configured HTTP port.
func (a *App) Start(ctx context.Context) error {
	if a.cfg.EnableProjector && a.projector != nil {
		go a.runProjector(ctx)
	}
	return a.server.ListenAndServe()
}

// runProjector runs an infinite loop, triggering the projector periodically
// according to the configuration. It exits when the provided context is done.
func (a *App) runProjector(ctx context.Context) {
	t := time.NewTicker(a.cfg.ProjectorEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = a.projector.RunOnce(ctx, a.cfg.ProjectorBatch)
		}
	}
}

// Shutdown gracefully stops the server and closes the underlying repository.
func (a *App) Shutdown(ctx context.Context) error {
	if a.repo != nil {
		_ = a.repo.Close()
	}
	return a.server.Shutdown(ctx)
}

// Flush runs the projector synchronously once, processing any pending outbox items.
// Useful for CLI commands that need to ensure writes reach the search index before exiting.
func (a *App) Flush(ctx context.Context) error {
	if a.projector != nil {
		_, err := a.projector.RunOnce(ctx, 1000)
		return err
	}
	return nil
}

// Handler returns the underlying HTTP handler for the context server API.
func (a *App) Handler() http.Handler {
	return a.server.Handler
}

// SymbolLocator returns the Meilisearch indexer for resolving codebase symbols.
func (a *App) SymbolLocator() *MeiliIndexer {
	return a.svc.(*AppService).searchBackend.(*MeiliIndexer)
}
