package env

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
)

// PikoEnv contains all PIKO_* environment variables for an environment.
type PikoEnv struct {
	Root     string         // PIKO_ROOT
	EnvName  string         // PIKO_ENV_NAME
	EnvPath  string         // PIKO_ENV_PATH
	Project  string         // PIKO_PROJECT
	Branch   string         // PIKO_BRANCH
	PortVars map[string]int // PIKO_<SERVICE>_PORT
}

// Build creates a PikoEnv from project, environment, and port allocations.
func Build(project *state.Project, env *state.Environment, allocations []ports.Allocation) *PikoEnv {
	portVars := make(map[string]int)
	for _, alloc := range allocations {
		varName := serviceToVarName(alloc.Service)
		portVars[varName] = alloc.HostPort
	}

	return &PikoEnv{
		Root:     project.RootPath,
		EnvName:  env.Name,
		EnvPath:  env.Path,
		Project:  env.DockerProject,
		Branch:   env.Branch,
		PortVars: portVars,
	}
}

// serviceToVarName converts a service name to a PIKO_*_PORT variable name.
// Example: "my-service" -> "PIKO_MY_SERVICE_PORT"
func serviceToVarName(service string) string {
	upper := strings.ToUpper(service)
	normalized := strings.ReplaceAll(upper, "-", "_")
	return "PIKO_" + normalized + "_PORT"
}

// ToEnvSlice returns the environment variables as a slice of "KEY=VALUE" strings.
func (e *PikoEnv) ToEnvSlice() []string {
	vars := []string{
		fmt.Sprintf("PIKO_ROOT=%s", e.Root),
		fmt.Sprintf("PIKO_ENV_NAME=%s", e.EnvName),
		fmt.Sprintf("PIKO_ENV_PATH=%s", e.EnvPath),
		fmt.Sprintf("PIKO_PROJECT=%s", e.Project),
		fmt.Sprintf("PIKO_BRANCH=%s", e.Branch),
	}

	for name, port := range e.PortVars {
		vars = append(vars, fmt.Sprintf("%s=%d", name, port))
	}

	return vars
}

// ToShellExport returns the environment variables in shell export format.
func (e *PikoEnv) ToShellExport() string {
	var lines []string
	for _, v := range e.ToEnvSlice() {
		lines = append(lines, v)
	}
	return strings.Join(lines, "\n")
}

// ToJSON returns the environment variables as JSON.
func (e *PikoEnv) ToJSON() ([]byte, error) {
	m := map[string]interface{}{
		"PIKO_ROOT":     e.Root,
		"PIKO_ENV_NAME": e.EnvName,
		"PIKO_ENV_PATH": e.EnvPath,
		"PIKO_PROJECT":  e.Project,
		"PIKO_BRANCH":   e.Branch,
	}
	for name, port := range e.PortVars {
		m[name] = port
	}
	return json.MarshalIndent(m, "", "  ")
}
