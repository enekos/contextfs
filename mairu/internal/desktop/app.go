package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"mairu/internal/agent"
	"mairu/internal/config"
	"mairu/internal/contextsrv"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application struct.
// All exported methods become callable from the frontend via window.go.desktop.App.
type App struct {
	ctx        context.Context
	svc        contextsrv.Service
	meili      *MeiliManager
	cfg        *config.Config
	agentsMu   sync.Mutex
	agents     map[string]*agent.Agent // session name → active agent
	meiliReady chan struct{}
}

// NewApp creates an uninitialized App. Call startup() to wire services.
func NewApp() *App {
	return &App{
		agents:     make(map[string]*agent.Agent),
		meiliReady: make(chan struct{}),
	}
}

// Startup is called by Wails when the app window is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("failed to get home dir", "error", err)
		return
	}

	cfg, err := config.Load(".")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return
	}
	a.cfg = cfg

	// Start managed Meilisearch
	a.meili = NewMeiliManager(filepath.Join(homeDir, ".mairu", "meilisearch"))

	if !a.meili.IsInstalled() {
		wailsRuntime.EventsEmit(ctx, "app:status", "Downloading Meilisearch...")
		if err := a.meili.Download(ctx, func(pct int) {
			wailsRuntime.EventsEmit(ctx, "app:download-progress", pct)
		}); err != nil {
			wailsRuntime.EventsEmit(ctx, "app:error", fmt.Sprintf("Failed to download Meilisearch: %v", err))
			return
		}
	}

	wailsRuntime.EventsEmit(ctx, "app:status", "Starting Meilisearch...")
	if err := a.meili.Start(ctx); err != nil {
		wailsRuntime.EventsEmit(ctx, "app:error", fmt.Sprintf("Failed to start Meilisearch: %v", err))
		return
	}

	// Wire the contextsrv.Service using the managed Meilisearch
	svcCfg := contextsrv.Config{
		Port:              0, // not used — no HTTP server in desktop mode
		SQLiteDSN:         cfg.Server.SQLiteDSN,
		MeiliURL:          a.meili.URL(),
		MeiliAPIKey:       a.meili.APIKey(),
		GeminiAPIKey:      cfg.API.GeminiAPIKey,
		ModerationEnabled: cfg.Server.ModerationEnabled,
		EmbeddingModel:    cfg.Embedding.Model,
		EmbeddingDim:      cfg.Embedding.Dimensions,
	}

	ctxApp, err := contextsrv.NewApp(svcCfg)
	if err != nil {
		wailsRuntime.EventsEmit(ctx, "app:error", fmt.Sprintf("Failed to init service: %v", err))
		return
	}
	a.svc = ctxApp.Service()

	close(a.meiliReady)

	wailsRuntime.EventsEmit(ctx, "app:ready", true)
}

// Shutdown is called by Wails when the window is closing.
func (a *App) Shutdown(ctx context.Context) {
	if a.meili != nil {
		if err := a.meili.Stop(); err != nil {
			slog.Error("failed to stop meilisearch", "error", err)
		}
	}
}

// Ping is a simple health check binding.
func (a *App) Ping() string {
	return "pong"
}
