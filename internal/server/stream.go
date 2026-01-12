package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gwuah/piko/internal/operations"
	"github.com/gwuah/piko/internal/stream"
)

type StreamCreateRequest struct {
	Action      string `json:"action"`
	Project     string `json:"project"`
	Environment string `json:"environment"`
	Branch      string `json:"branch"`
}

func (s *Server) handleCreateEnvironmentStream(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.PathValue("projectID")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := s.db.GetProjectByID(projectID)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("failed to read create request: %v", err)
		return
	}

	var req StreamCreateRequest
	if err := json.Unmarshal(message, &req); err != nil {
		stream.SendError(conn, "invalid request format")
		return
	}

	if req.Environment == "" {
		stream.SendError(conn, "environment name is required")
		return
	}

	factory := stream.NewWriterFactory(conn, os.Stdout)
	gitStdout, gitStderr := factory.Git()
	dockerStdout, dockerStderr := factory.Docker()
	prepareStdout, prepareStderr := factory.Prepare()
	setupStdout, setupStderr := factory.Setup()

	pikoWriter := factory.Piko()
	pikoLogger := &operations.WriterLogger{Out: pikoWriter, Err: pikoWriter}

	result, err := operations.CreateEnvironment(operations.CreateEnvironmentOptions{
		DB:      s.db,
		Project: project,
		Name:    req.Environment,
		Branch:  req.Branch,
		Logger:  pikoLogger,
		Output: &operations.OutputWriters{
			GitStdout:     gitStdout,
			GitStderr:     gitStderr,
			DockerStdout:  dockerStdout,
			DockerStderr:  dockerStderr,
			PrepareStdout: prepareStdout,
			PrepareStderr: prepareStderr,
			SetupStdout:   setupStdout,
			SetupStderr:   setupStderr,
		},
	})

	gitStdout.Flush()
	gitStderr.Flush()
	dockerStdout.Flush()
	dockerStderr.Flush()
	prepareStdout.Flush()
	prepareStderr.Flush()
	setupStdout.Flush()
	setupStderr.Flush()
	pikoWriter.Flush()

	if err != nil {
		stream.SendError(conn, err.Error())
		return
	}

	mode := "docker"
	status := "running"
	if result.IsSimple {
		mode = "simple"
		status = "simple"
	}

	s.broadcastStateChange("env_created", project.ID, result.Environment.Name)

	stream.SendComplete(conn, &stream.Environment{
		ID:     result.Environment.ID,
		Name:   result.Environment.Name,
		Branch: result.Environment.Branch,
		Path:   result.Environment.Path,
		Mode:   mode,
		Status: status,
	})
}

type StreamDestroyRequest struct {
	Action        string `json:"action"`
	Environment   string `json:"environment"`
	RemoveVolumes bool   `json:"remove_volumes"`
	DeleteBranch  bool   `json:"delete_branch"`
}

func (s *Server) handleDestroyEnvironmentStream(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.PathValue("projectID")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := s.db.GetProjectByID(projectID)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	name := r.PathValue("name")
	environment, err := s.db.GetEnvironmentByName(projectID, name)
	if err != nil {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("failed to read destroy request: %v", err)
		return
	}

	var req StreamDestroyRequest
	if err := json.Unmarshal(message, &req); err != nil {
		stream.SendError(conn, "invalid request format")
		return
	}

	factory := stream.NewWriterFactory(conn, os.Stdout)
	destroyStdout, destroyStderr := factory.Destroy()
	dockerStdout, dockerStderr := factory.Docker()
	pikoWriter := factory.Piko()
	pikoLogger := &operations.WriterLogger{Out: pikoWriter, Err: pikoWriter}

	err = operations.DestroyEnvironment(operations.DestroyEnvironmentOptions{
		DB:            s.db,
		Project:       project,
		Environment:   environment,
		RemoveVolumes: req.RemoveVolumes,
		DeleteBranch:  req.DeleteBranch,
		Logger:        pikoLogger,
		Output: &operations.DestroyOutputWriters{
			DestroyStdout: destroyStdout,
			DestroyStderr: destroyStderr,
			DockerStdout:  dockerStdout,
			DockerStderr:  dockerStderr,
		},
	})

	destroyStdout.Flush()
	destroyStderr.Flush()
	dockerStdout.Flush()
	dockerStderr.Flush()
	pikoWriter.Flush()

	if err != nil {
		stream.SendError(conn, err.Error())
		return
	}

	s.broadcastStateChange("env_deleted", project.ID, name)
	stream.SendComplete(conn, nil)
}

type StreamLogger struct {
	writer *stream.StreamWriter
}

func (l *StreamLogger) Info(msg string) {
	fmt.Fprintln(l.writer, msg)
}

func (l *StreamLogger) Infof(format string, args ...any) {
	fmt.Fprintf(l.writer, format+"\n", args...)
}

func (l *StreamLogger) Warn(msg string) {
	fmt.Fprintln(l.writer, "Warning: "+msg)
}

func (l *StreamLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(l.writer, "Warning: "+format+"\n", args...)
}
