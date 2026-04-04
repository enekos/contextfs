package contextsrv

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Port            int
	PostgresDSN     string
	MeiliURL        string
	MeiliAPIKey     string
	AuthToken       string
	EnableProjector bool
	ProjectorEvery  time.Duration
	ProjectorBatch  int
}

type App struct {
	cfg       Config
	repo      *PostgresRepository
	projector *Projector
	server    *http.Server
}

func NewApp(cfg Config) (*App, error) {
	if cfg.Port == 0 {
		cfg.Port = 8788
	}
	if cfg.PostgresDSN == "" {
		cfg.PostgresDSN = os.Getenv("CONTEXT_SERVER_POSTGRES_DSN")
	}
	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("CONTEXT_SERVER_POSTGRES_DSN is required")
	}
	if cfg.MeiliURL == "" {
		cfg.MeiliURL = os.Getenv("MEILI_URL")
	}
	if cfg.MeiliURL == "" {
		cfg.MeiliURL = "http://localhost:7700"
	}
	if cfg.ProjectorEvery <= 0 {
		cfg.ProjectorEvery = 3 * time.Second
	}
	if cfg.ProjectorBatch <= 0 {
		cfg.ProjectorBatch = 50
	}

	repo, err := NewPostgresRepository(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	indexer := NewMeiliIndexer(cfg.MeiliURL, cfg.MeiliAPIKey)
	_ = indexer.EnsureIndexes()
	svc := NewServiceWithSearch(repo, indexer)
	handler := NewHandler(svc, cfg.AuthToken)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return &App{
		cfg:       cfg,
		repo:      repo,
		projector: NewProjector(repo, indexer),
		server:    srv,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	if a.cfg.EnableProjector {
		go a.runProjector(ctx)
	}
	return a.server.ListenAndServe()
}

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

func (a *App) Shutdown(ctx context.Context) error {
	_ = a.repo.Close()
	return a.server.Shutdown(ctx)
}
