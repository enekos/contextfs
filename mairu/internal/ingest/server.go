package ingest

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
)

// BashRepo is the persistence interface for bash history records.
// Task 4 will wire a concrete implementation.
type BashRepo interface {
	InsertBashHistory(ctx context.Context, project, command string, exitCode, durationMs int, output string) error
}

// redactorIface is a placeholder that lets Task 4 inject *redact.Redactor
// without this package importing it directly.
type redactorIface interface {
	// Intentionally empty; Task 4 will add method(s) here.
}

// Server listens on a Unix socket and dispatches ingest records.
type Server struct {
	path     string
	repo     BashRepo
	redactor redactorIface

	// testHook, if non-nil, replaces the default processRecord path.
	// It is unexported so only tests within this package can set it.
	testHook func(context.Context, Record)

	wg sync.WaitGroup
}

// NewServer constructs a Server that will listen at path.
func NewServer(path string, repo BashRepo, redactor redactorIface) *Server {
	return &Server{path: path, repo: repo, redactor: redactor}
}

// Run listens on the Unix socket at s.path and serves record streams until
// ctx is cancelled or an unrecoverable error occurs. On shutdown it closes
// the listener and removes the socket file.
func (s *Server) Run(ctx context.Context) error {
	// Remove any stale socket file from a previous run.
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}

	// Restrict access to the current user only.
	if err := os.Chmod(s.path, 0o600); err != nil {
		listener.Close()
		return err
	}

	acceptErrCh := make(chan error, 1)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					acceptErrCh <- nil
					return
				}
				acceptErrCh <- err
				return
			}
			s.wg.Add(1)
			go s.handleConn(ctx, conn)
		}
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown path.
	case err := <-acceptErrCh:
		// Unexpected accept error — clean up and propagate.
		listener.Close()
		s.wg.Wait()
		os.Remove(s.path) //nolint:errcheck
		return err
	}

	listener.Close()
	s.wg.Wait()
	os.Remove(s.path) //nolint:errcheck
	return nil
}

// handleConn reads records from conn until EOF or a decode error, then exits.
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	r := bufio.NewReader(conn)
	for {
		rec, err := Decode(r)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			slog.Debug("ingest: decode error", "err", err)
			return
		}
		s.processRecord(ctx, rec)
	}
}

// processRecord dispatches rec. If testHook is set it takes priority;
// otherwise this is a no-op until Task 4 wires the real pipeline.
func (s *Server) processRecord(ctx context.Context, rec Record) {
	if s.testHook != nil {
		s.testHook(ctx, rec)
		return
	}
	// Task 4 will call s.redactor + s.repo here.
}
