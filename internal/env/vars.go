package env

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
)

type PikoEnv struct {
	Root     string
	EnvName  string
	EnvID    int64
	EnvPath  string
	DataDir  string
	Project  string
	Branch   string
	PortVars map[string]int
}

func Build(project *state.Project, env *state.Environment, allocations []ports.Allocation) *PikoEnv {
	portVars := make(map[string]int)
	for _, alloc := range allocations {
		varName := serviceToVarName(alloc.Service)
		portVars[varName] = alloc.HostPort
	}

	dataDir := filepath.Join(project.RootPath, ".piko", "data", env.Name)

	return &PikoEnv{
		Root:     project.RootPath,
		EnvName:  env.Name,
		EnvID:    env.ID,
		EnvPath:  env.Path,
		DataDir:  dataDir,
		Project:  project.Name,
		Branch:   env.Branch,
		PortVars: portVars,
	}
}

func serviceToVarName(service string) string {
	upper := strings.ToUpper(service)
	normalized := strings.ReplaceAll(upper, "-", "_")
	return "PIKO_" + normalized + "_PORT"
}

func (e *PikoEnv) ToEnvSlice() []string {
	vars := []string{
		fmt.Sprintf("PIKO_ROOT=%s", e.Root),
		fmt.Sprintf("PIKO_ENV_NAME=%s", e.EnvName),
		fmt.Sprintf("PIKO_ENV_ID=%d", e.EnvID),
		fmt.Sprintf("PIKO_ENV_PATH=%s", e.EnvPath),
		fmt.Sprintf("PIKO_DATA_DIR=%s", e.DataDir),
		fmt.Sprintf("PIKO_PROJECT=%s", e.Project),
		fmt.Sprintf("PIKO_BRANCH=%s", e.Branch),
	}

	for name, port := range e.PortVars {
		vars = append(vars, fmt.Sprintf("%s=%d", name, port))
	}

	return vars
}

func (e *PikoEnv) ToShellExport() string {
	var lines []string
	for _, v := range e.ToEnvSlice() {
		lines = append(lines, v)
	}
	return strings.Join(lines, "\n")
}

func (e *PikoEnv) ToJSON() ([]byte, error) {
	m := map[string]interface{}{
		"PIKO_ROOT":     e.Root,
		"PIKO_ENV_NAME": e.EnvName,
		"PIKO_ENV_ID":   e.EnvID,
		"PIKO_ENV_PATH": e.EnvPath,
		"PIKO_DATA_DIR": e.DataDir,
		"PIKO_PROJECT":  e.Project,
		"PIKO_BRANCH":   e.Branch,
	}
	for name, port := range e.PortVars {
		m[name] = port
	}
	return json.MarshalIndent(m, "", "  ")
}
