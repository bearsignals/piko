package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/git"
	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
)

type ProjectResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Initialized bool   `json:"initialized"`
}

type EnvironmentResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
}

type CreateRequest struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

type SuccessResponse struct {
	Success     bool                 `json:"success"`
	Environment *EnvironmentResponse `json:"environment,omitempty"`
	Error       string               `json:"error,omitempty"`
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.db.GetProject()
	if err != nil {
		writeJSON(w, http.StatusOK, ProjectResponse{Initialized: false})
		return
	}

	writeJSON(w, http.StatusOK, ProjectResponse{
		Name:        project.Name,
		Path:        project.RootPath,
		Initialized: true,
	})
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	environments, err := s.db.ListEnvironments()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	response := make([]EnvironmentResponse, 0, len(environments))
	for _, e := range environments {
		status := docker.GetProjectStatus(e.Path, e.DockerProject)
		response = append(response, EnvironmentResponse{
			Name:   e.Name,
			Status: string(status),
			Branch: e.Branch,
			Path:   e.Path,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SuccessResponse{Success: false, Error: "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, SuccessResponse{Success: false, Error: "name is required"})
		return
	}

	exists, err := s.db.EnvironmentExists(req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q already exists", req.Name)})
		return
	}

	project, err := s.db.GetProject()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	worktreesDir := filepath.Join(s.rootDir, ".piko", "worktrees")

	wtOpts := git.WorktreeOptions{
		Name:       req.Name,
		BasePath:   worktreesDir,
		BranchName: req.Branch,
	}
	wt, err := git.CreateWorktree(wtOpts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	composeConfig, err := docker.ParseComposeConfig(wt.Path)
	if err != nil {
		git.RemoveWorktree(wt.Path)
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	dockerProject := fmt.Sprintf("piko-%s-%s", project.Name, req.Name)
	environment := &state.Environment{
		ProjectID:     project.ID,
		Name:          req.Name,
		Branch:        wt.Branch,
		Path:          wt.Path,
		DockerProject: dockerProject,
	}
	envID, err := s.db.InsertEnvironment(environment)
	if err != nil {
		git.RemoveWorktree(wt.Path)
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}
	environment.ID = envID

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(envID, servicePorts)

	override := docker.GenerateOverride(project.Name, req.Name, allocations)
	overridePath := filepath.Join(wt.Path, "docker-compose.piko.yml")
	if err := docker.WriteOverrideFile(overridePath, override); err != nil {
		s.db.DeleteEnvironment(req.Name)
		git.RemoveWorktree(wt.Path)
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	composeCmd := exec.Command("docker", "compose",
		"-p", dockerProject,
		"-f", "docker-compose.yml",
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = wt.Path

	if err := composeCmd.Run(); err != nil {
		s.db.DeleteEnvironment(req.Name)
		git.RemoveWorktree(wt.Path)
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: "failed to start containers"})
		return
	}

	cfg, _ := config.Load(s.rootDir)
	if cfg != nil && cfg.Scripts.Setup != "" {
		pikoEnv := env.Build(project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())
		runner.RunSetup(cfg.Scripts.Setup)
	}

	writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Environment: &EnvironmentResponse{
			Name:   environment.Name,
			Status: "running",
			Branch: environment.Branch,
			Path:   environment.Path,
		},
	})
}

func (s *Server) handleOpenInEditor(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	environment, err := s.db.GetEnvironmentByName(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q not found", name)})
		return
	}

	editors := []string{"cursor", "code", "vim"}
	var opened bool

	for _, editor := range editors {
		cmd := exec.Command(editor, environment.Path)
		if err := cmd.Start(); err == nil {
			opened = true
			break
		}
	}

	if !opened {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: "no editor found (tried: cursor, code, vim)"})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func (s *Server) handleUp(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	project, err := s.db.GetProject()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	environment, err := s.db.GetEnvironmentByName(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q not found", name)})
		return
	}

	composeConfig, err := docker.ParseComposeConfig(environment.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(environment.ID, servicePorts)

	override := docker.GenerateOverride(project.Name, name, allocations)
	overridePath := filepath.Join(environment.Path, "docker-compose.piko.yml")
	docker.WriteOverrideFile(overridePath, override)

	cmd := exec.Command("docker", "compose",
		"-p", environment.DockerProject,
		"-f", "docker-compose.yml",
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	cmd.Dir = environment.Path

	if err := cmd.Run(); err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: "failed to start containers"})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func (s *Server) handleDown(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	environment, err := s.db.GetEnvironmentByName(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q not found", name)})
		return
	}

	cmd := exec.Command("docker", "compose", "-p", environment.DockerProject, "down")
	cmd.Dir = environment.Path

	if err := cmd.Run(); err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: "failed to stop containers"})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
