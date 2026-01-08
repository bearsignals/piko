package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gwuah/piko/internal/state"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	port    int
	db      *state.DB
	rootDir string
	server  *http.Server
}

func New(port int, db *state.DB, rootDir string) *Server {
	return &Server{
		port:    port,
		db:      db,
		rootDir: rootDir,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/project", s.handleGetProject)
	mux.HandleFunc("GET /api/environments", s.handleListEnvironments)
	mux.HandleFunc("POST /api/environments", s.handleCreateEnvironment)
	mux.HandleFunc("POST /api/environments/{name}/open", s.handleOpenInEditor)
	mux.HandleFunc("POST /api/environments/{name}/up", s.handleUp)
	mux.HandleFunc("POST /api/environments/{name}/down", s.handleDown)

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to get static files: %w", err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}()

	fmt.Printf("→ Piko server running at http://localhost:%d\n", s.port)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
