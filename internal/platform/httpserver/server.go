package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Config controls the listener and its timeouts.
type Config struct {
	Addr            string // For example ":8080".
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Server runs an http.Server with graceful shutdown.
type Server struct {
	srv             *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewServer validates cfg and returns a server that serves handler. A nil logger falls back to slog.Default().
func NewServer(cfg Config, handler http.Handler, logger *slog.Logger) (*Server, error) {
	if cfg.Addr == "" {
		return nil, errors.New("httpserver: address is required")
	}
	if cfg.ReadTimeout <= 0 || cfg.WriteTimeout <= 0 || cfg.ShutdownTimeout <= 0 {
		return nil, errors.New("httpserver: read, write and shutdown timeouts must be positive")
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		srv: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

// Run serves until ctx is cancelled, then shuts down gracefully within the shutdown timeout.
// It returns an error if the listener fails or if shutdown does not finish in time.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http server listening", slog.String("addr", s.srv.Addr))
		errCh <- s.srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// The listener stopped on its own, for example because the port is already in use.
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpserver: listen: %w", err)
	case <-ctx.Done():
	}

	s.logger.Info("http server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		_ = s.srv.Close()
		return fmt.Errorf("httpserver: shutdown: %w", err)
	}
	<-errCh
	return nil
}
