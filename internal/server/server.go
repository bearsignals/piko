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
	port   int
	db     *state.DB
	server *http.Server
}

func New(port int, db *state.DB) *Server {
	return &Server{
		port: port,
		db:   db,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("GET /api/projects/{projectID}/environments", s.handleListEnvironments)
	mux.HandleFunc("GET /api/projects/{projectID}/environments/{name}", s.handleGetEnvironment)
	mux.HandleFunc("POST /api/projects/{projectID}/environments", s.handleCreateEnvironment)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/open", s.handleOpenInEditor)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/up", s.handleUp)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/down", s.handleDown)
	mux.HandleFunc("DELETE /api/projects/{projectID}/environments/{name}", s.handleDestroyEnvironment)

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to get static files: %w", err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      http.TimeoutHandler(mux, 60*time.Second, "request timeout"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
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
