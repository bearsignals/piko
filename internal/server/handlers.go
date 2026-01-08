package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/operations"
	"github.com/gwuah/piko/internal/run"
	"github.com/gwuah/piko/internal/state"
)

const handlerDockerTimeout = 10 * time.Second

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
	Mode       string          `json:"mode"`
	DataDir    string          `json:"dataDir,omitempty"`
	EnvID      int64           `json:"envId,omitempty"`
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
		isSimpleMode := e.DockerProject == ""

		envResp := EnvironmentResponse{
			Name:   e.Name,
			Branch: e.Branch,
			Path:   e.Path,
			EnvID:  e.ID,
		}

		if isSimpleMode {
			envResp.Mode = "simple"
			envResp.Status = "simple"
			envResp.DataDir = filepath.Join(project.RootPath, ".piko", "data", e.Name)
		} else {
			envResp.Mode = "docker"
			composeDir := e.Path
			if project.ComposeDir != "" {
				composeDir = filepath.Join(e.Path, project.ComposeDir)
			}
			envResp.Status = string(docker.GetProjectStatus(composeDir, e.DockerProject))
			envResp.Ports, envResp.Containers, envResp.Running, envResp.Total = s.getEnvironmentDetails(composeDir, e.DockerProject)
		}

		response = append(response, envResp)
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

	isSimpleMode := environment.DockerProject == ""

	envResp := EnvironmentResponse{
		Name:   environment.Name,
		Branch: environment.Branch,
		Path:   environment.Path,
		EnvID:  environment.ID,
	}

	if isSimpleMode {
		envResp.Mode = "simple"
		envResp.Status = "simple"
		envResp.DataDir = filepath.Join(project.RootPath, ".piko", "data", environment.Name)
	} else {
		envResp.Mode = "docker"
		composeDir := environment.Path
		if project.ComposeDir != "" {
			composeDir = filepath.Join(environment.Path, project.ComposeDir)
		}
		envResp.Status = string(docker.GetProjectStatus(composeDir, environment.DockerProject))
		envResp.Ports, envResp.Containers, envResp.Running, envResp.Total = s.getEnvironmentDetails(composeDir, environment.DockerProject)
	}

	writeJSON(w, http.StatusOK, envResp)
}

func (s *Server) getEnvironmentDetails(composeDir, dockerProject string) ([]PortMapping, []ContainerInfo, int, int) {
	var portMappings []PortMapping
	var containers []ContainerInfo
	running := 0
	total := 0

	output, err := run.Command("docker", "compose", "-p", dockerProject, "ps", "--format", "json").
		Dir(composeDir).
		Timeout(handlerDockerTimeout).
		Output()
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

	result, err := operations.CreateEnvironment(operations.CreateEnvironmentOptions{
		DB:      s.db,
		Project: project,
		Name:    req.Name,
		Branch:  req.Branch,
		Logger:  &operations.SilentLogger{},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	status := "running"
	mode := "docker"
	if result.IsSimple {
		status = "simple"
		mode = "simple"
	}

	writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Environment: &EnvironmentResponse{
			Name:    result.Environment.Name,
			Status:  status,
			Branch:  result.Environment.Branch,
			Path:    result.Environment.Path,
			Mode:    mode,
			DataDir: result.DataDir,
			EnvID:   result.Environment.ID,
		},
	})
}

func (s *Server) handleDestroyEnvironment(w http.ResponseWriter, r *http.Request) {
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

	err = operations.DestroyEnvironment(operations.DestroyEnvironmentOptions{
		DB:            s.db,
		Project:       project,
		Environment:   environment,
		RemoveVolumes: false,
		Logger:        &operations.SilentLogger{},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
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

	err = operations.UpEnvironment(operations.UpEnvironmentOptions{
		DB:          s.db,
		Project:     project,
		Environment: environment,
		Logger:      &operations.SilentLogger{},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
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

	err = operations.DownEnvironment(operations.DownEnvironmentOptions{
		DB:          s.db,
		Project:     project,
		Environment: environment,
		Logger:      &operations.SilentLogger{},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
