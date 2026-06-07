// Package adminapi wires the HTTP server.
package adminapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/woodleighschool/woodstar/internal/config"
)

// Server owns the HTTP listener and router.
type Server struct {
	httpServer *http.Server
	config     config.Config
	logger     *slog.Logger
	version    string
}

// NewServer returns an HTTP server.
func NewServer(deps Dependencies) *Server {
	server := &Server{
		config:  deps.Runtime.Config,
		logger:  deps.Runtime.Logger,
		version: deps.Runtime.Version,
	}
	server.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", deps.Runtime.Config.Host, deps.Runtime.Config.Port),
		Handler:           routes(deps),
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       180 * time.Second,
	}
	return server
}

// Addr returns the configured HTTP listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Serve starts the HTTP server on listener and blocks until shutdown or failure.
func (s *Server) Serve(listener net.Listener) error {
	s.logger.Info(
		"starting woodstar",
		"component", "server",
		"operation", "start",
		"addr", s.httpServer.Addr,
		"public_url", s.config.PublicURL,
		"version", s.version,
	)
	if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "stopping woodstar", "component", "server", "operation", "shutdown")
	return s.httpServer.Shutdown(ctx)
}
