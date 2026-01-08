package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/git"
	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
)

type ProjectResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Initialized bool   `json:"initialized"`
}

type PortMapping struct {
	Service       string `json:"service"`
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	URL           string `json:"url,omitempty"`
}

type ContainerInfo struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Health string `json:"health,omitempty"`
}

type EnvironmentResponse struct {
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Branch     string          `json:"branch"`
	Path       string          `json:"path"`
	Ports      []PortMapping   `json:"ports,omitempty"`
	Containers []ContainerInfo `json:"containers,omitempty"`
	Running    int             `json:"running"`
	Total      int             `json:"total"`
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

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.db.ListProjects()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	response := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		response = append(response, ProjectResponse{
			ID:          p.ID,
			Name:        p.Name,
			Path:        p.RootPath,
			Initialized: true,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getProjectFromPath(r *http.Request) (*state.Project, error) {
	projectIDStr := r.PathValue("projectID")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID")
	}
	return s.db.GetProjectByID(projectID)
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectFromPath(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	environments, err := s.db.ListEnvironmentsByProject(project.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	response := make([]EnvironmentResponse, 0, len(environments))
	for _, e := range environments {
		composeDir := e.Path
		if project.ComposeDir != "" {
			composeDir = filepath.Join(e.Path, project.ComposeDir)
		}
		status := docker.GetProjectStatus(composeDir, e.DockerProject)

		portMappings, containers, running, total := s.getEnvironmentDetails(composeDir, e.DockerProject)

		response = append(response, EnvironmentResponse{
			Name:       e.Name,
			Status:     string(status),
			Branch:     e.Branch,
			Path:       e.Path,
			Ports:      portMappings,
			Containers: containers,
			Running:    running,
			Total:      total,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	project, err := s.getProjectFromPath(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	environment, err := s.db.GetEnvironmentByName(project.ID, name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q not found", name)})
		return
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	status := docker.GetProjectStatus(composeDir, environment.DockerProject)
	portMappings, containers, running, total := s.getEnvironmentDetails(composeDir, environment.DockerProject)

	writeJSON(w, http.StatusOK, EnvironmentResponse{
		Name:       environment.Name,
		Status:     string(status),
		Branch:     environment.Branch,
		Path:       environment.Path,
		Ports:      portMappings,
		Containers: containers,
		Running:    running,
		Total:      total,
	})
}

func (s *Server) getEnvironmentDetails(composeDir, dockerProject string) ([]PortMapping, []ContainerInfo, int, int) {
	var portMappings []PortMapping
	var containers []ContainerInfo
	running := 0
	total := 0

	cmd := exec.Command("docker", "compose", "-p", dockerProject, "ps", "--format", "json")
	cmd.Dir = composeDir
	output, err := cmd.Output()
	if err != nil {
		return portMappings, containers, running, total
	}

	seenPorts := make(map[string]bool)

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		var c struct {
			Service    string `json:"Service"`
			Name       string `json:"Name"`
			State      string `json:"State"`
			Health     string `json:"Health"`
			Publishers []struct {
				TargetPort    int `json:"TargetPort"`
				PublishedPort int `json:"PublishedPort"`
			} `json:"Publishers"`
		}
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}

		total++
		if c.State == "running" {
			running++
		}

		containers = append(containers, ContainerInfo{
			Name:   c.Name,
			State:  c.State,
			Health: c.Health,
		})

		for _, pub := range c.Publishers {
			if pub.PublishedPort == 0 {
				continue
			}
			key := fmt.Sprintf("%s:%d", c.Service, pub.TargetPort)
			if seenPorts[key] {
				continue
			}
			seenPorts[key] = true

			pm := PortMapping{
				Service:       c.Service,
				ContainerPort: pub.TargetPort,
				HostPort:      pub.PublishedPort,
			}
			if isHTTPPort(pub.TargetPort) {
				pm.URL = fmt.Sprintf("http://localhost:%d", pub.PublishedPort)
			}
			portMappings = append(portMappings, pm)
		}
	}

	return portMappings, containers, running, total
}

func isHTTPPort(port int) bool {
	httpPorts := []int{80, 443, 3000, 3001, 4000, 5000, 5173, 8000, 8080, 8081, 8888, 9000}
	for _, p := range httpPorts {
		if port == p {
			return true
		}
	}
	return false
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

	project, err := s.getProjectFromPath(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	exists, err := s.db.EnvironmentExists(project.ID, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q already exists", req.Name)})
		return
	}

	worktreesDir := filepath.Join(project.RootPath, ".piko", "worktrees")

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

	composeDir := wt.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(wt.Path, project.ComposeDir)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
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

	composeProject := composeConfig.Project()
	docker.ApplyOverrides(composeProject, project.Name, req.Name, allocations)
	pikoComposePath := filepath.Join(composeDir, "docker-compose.piko.yml")
	if err := docker.WriteProjectFile(pikoComposePath, composeProject); err != nil {
		s.db.DeleteEnvironment(project.ID, req.Name)
		git.RemoveWorktree(wt.Path)
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	composeCmd := exec.Command("docker", "compose",
		"-p", dockerProject,
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = composeDir

	if err := composeCmd.Run(); err != nil {
		s.db.DeleteEnvironment(project.ID, req.Name)
		git.RemoveWorktree(wt.Path)
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: "failed to start containers"})
		return
	}

	cfg, _ := config.Load(project.RootPath)
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

	project, err := s.getProjectFromPath(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	environment, err := s.db.GetEnvironmentByName(project.ID, name)
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

	project, err := s.getProjectFromPath(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	environment, err := s.db.GetEnvironmentByName(project.ID, name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q not found", name)})
		return
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(environment.ID, servicePorts)

	composeProject := composeConfig.Project()
	docker.ApplyOverrides(composeProject, project.Name, name, allocations)
	pikoComposePath := filepath.Join(composeDir, "docker-compose.piko.yml")
	docker.WriteProjectFile(pikoComposePath, composeProject)

	cmd := exec.Command("docker", "compose",
		"-p", environment.DockerProject,
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	cmd.Dir = composeDir

	if err := cmd.Run(); err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: "failed to start containers"})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func (s *Server) handleDown(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	project, err := s.getProjectFromPath(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	environment, err := s.db.GetEnvironmentByName(project.ID, name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: fmt.Sprintf("environment %q not found", name)})
		return
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	cmd := exec.Command("docker", "compose", "-p", environment.DockerProject, "down")
	cmd.Dir = composeDir

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
