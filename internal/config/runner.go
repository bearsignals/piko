package config

import (
	"fmt"
	"os"
	"os/exec"
)

// ScriptRunner executes lifecycle scripts with the appropriate environment.
type ScriptRunner struct {
	WorkDir string
	Env     []string
}

// NewScriptRunner creates a new script runner.
func NewScriptRunner(workDir string, env []string) *ScriptRunner {
	return &ScriptRunner{
		WorkDir: workDir,
		Env:     env,
	}
}

// RunSetup executes the setup script.
// Returns nil if the script is empty.
func (r *ScriptRunner) RunSetup(script string) error {
	if script == "" {
		return nil
	}
	return r.run(script)
}

// RunDestroy executes the destroy script.
// Returns nil if the script is empty.
func (r *ScriptRunner) RunDestroy(script string) error {
	if script == "" {
		return nil
	}
	return r.run(script)
}

func (r *ScriptRunner) run(script string) error {
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = r.WorkDir
	cmd.Env = append(os.Environ(), r.Env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w", err)
	}
	return nil
}
