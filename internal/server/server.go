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
	"github.com/gwuah/piko/internal/version"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	port    int
	db      *state.DB
	server  *http.Server
	hub     *Hub
	devMode bool
}

func New(port int, db *state.DB) *Server {
	return &Server{
		port:    port,
		db:      db,
		hub:     NewHub(),
		devMode: os.Getenv("PIKO_DEV") == "1",
	}
}

func (s *Server) Start() error {
	go s.hub.Run()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/orchestra/ws", s.handleOrchestraWS)
	mux.HandleFunc("GET /api/orchestra/notifications", s.handleOrchestraList)
	mux.HandleFunc("POST /api/orchestra/notify", s.handleOrchestraNotify)
	mux.HandleFunc("POST /api/orchestra/respond", s.handleOrchestraRespond)
	mux.HandleFunc("DELETE /api/orchestra/notifications/{id}", s.handleOrchestraDismiss)

	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("GET /api/projects/{projectID}/environments", s.handleListEnvironments)
	mux.HandleFunc("GET /api/projects/{projectID}/environments/{name}", s.handleGetEnvironment)
	mux.HandleFunc("POST /api/projects/{projectID}/environments", s.handleCreateEnvironment)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/open", s.handleOpenInEditor)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/up", s.handleUp)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/down", s.handleDown)
	mux.HandleFunc("POST /api/projects/{projectID}/environments/{name}/restart", s.handleRestart)
	mux.HandleFunc("DELETE /api/projects/{projectID}/environments/{name}", s.handleDestroyEnvironment)

	if s.devMode {
		mux.Handle("GET /", http.FileServer(http.Dir("internal/server/static")))
		fmt.Println("→ Dev mode: serving static files from disk")
	} else {
		staticFS, err := fs.Sub(staticFiles, "static")
		if err != nil {
			return fmt.Errorf("failed to get static files: %w", err)
		}
		mux.Handle("GET /", http.FileServer(http.FS(staticFS)))
	}

	timeoutHandler := http.TimeoutHandler(mux, 60*time.Second, "request timeout")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/orchestra/ws" {
			mux.ServeHTTP(w, r)
			return
		}
		timeoutHandler.ServeHTTP(w, r)
	})

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
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

	fmt.Printf("→ Piko server (%s) running at http://localhost:%d\n", version.Info(), s.port)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
