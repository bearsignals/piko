package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func SessionName(projectName, envName string) string {
	return fmt.Sprintf("piko/%s/%s", projectName, envName)
}

func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

func SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

func CreateSession(sessionName, workDir string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create session: %s: %w", string(output), err)
	}
	return nil
}

func RenameWindow(sessionName, windowName string) error {
	cmd := exec.Command("tmux", "rename-window", "-t", sessionName, windowName)
	return cmd.Run()
}

func NewWindow(sessionName, windowName, workDir, command string) error {
	cmd := exec.Command("tmux", "new-window", "-t", sessionName, "-n", windowName, "-c", workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create window: %s: %w", string(output), err)
	}

	if command != "" {
		sendCmd := exec.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), command, "Enter")
		sendCmd.Run()
	}

	return nil
}

func SendKeys(sessionName, windowName, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), keys, "Enter")
	return cmd.Run()
}

func Attach(sessionName string) error {
	cmd := exec.Command("tmux", "attach", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Switch(sessionName string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
	return cmd.Run()
}

func KillSession(sessionName string) error {
	if !SessionExists(sessionName) {
		return nil
	}
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

func ListPikoSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(line, "piko/") {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

type SessionConfig struct {
	SessionName   string
	WorkDir       string
	DockerProject string
	Services      []string
	Shells        map[string]string
}

func CreateFullSession(cfg SessionConfig) error {
	if err := CreateSession(cfg.SessionName, cfg.WorkDir); err != nil {
		return err
	}

	if err := RenameWindow(cfg.SessionName, "shell"); err != nil {
		return err
	}

	for _, service := range cfg.Services {
		shell := "sh"
		if s, ok := cfg.Shells[service]; ok {
			shell = s
		}

		windowCmd := fmt.Sprintf("docker compose -p %s exec %s %s", cfg.DockerProject, service, shell)
		if err := NewWindow(cfg.SessionName, service, cfg.WorkDir, windowCmd); err != nil {
			continue
		}
	}

	logsCmd := fmt.Sprintf("docker compose -p %s logs -f", cfg.DockerProject)
	NewWindow(cfg.SessionName, "logs", cfg.WorkDir, logsCmd)

	return nil
}
