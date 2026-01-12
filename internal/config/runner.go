package config

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type ScriptRunner struct {
	WorkDir string
	Env     []string
	Stdout  io.Writer
	Stderr  io.Writer
}

func NewScriptRunner(workDir string, env []string) *ScriptRunner {
	return &ScriptRunner{
		WorkDir: workDir,
		Env:     env,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

func (r *ScriptRunner) WithOutput(stdout, stderr io.Writer) *ScriptRunner {
	r.Stdout = stdout
	r.Stderr = stderr
	return r
}

func (r *ScriptRunner) RunPrepare(script string) error {
	if script == "" {
		return nil
	}
	return r.run(script)
}

func (r *ScriptRunner) RunSetup(script string) error {
	if script == "" {
		return nil
	}
	return r.run(script)
}

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
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w", err)
	}
	return nil
}
