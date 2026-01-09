package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gwuah/piko/internal/run"
)

const tmuxTimeout = 5 * time.Second

func SessionName(projectName, envName string) string {
	return fmt.Sprintf("piko/%s/%s", projectName, envName)
}

func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

func SessionExists(sessionName string) bool {
	err := run.Command("tmux", "has-session", "-t", sessionName).
		Timeout(tmuxTimeout).
		Run()
	return err == nil
}

func CreateSession(sessionName, workDir string) error {
	output, err := run.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", workDir).
		Timeout(tmuxTimeout).
		CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create session: %s: %w", string(output), err)
	}
	return nil
}

func RenameWindow(sessionName, windowName string) error {
	return run.Command("tmux", "rename-window", "-t", sessionName, windowName).
		Timeout(tmuxTimeout).
		Run()
}

func NewWindow(sessionName, windowName, workDir, command string) error {
	output, err := run.Command("tmux", "new-window", "-t", sessionName, "-n", windowName, "-c", workDir).
		Timeout(tmuxTimeout).
		CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create window: %s: %w", string(output), err)
	}

	if command != "" {
		run.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), command, "Enter").
			Timeout(tmuxTimeout).
			Run()
	}

	return nil
}

func SendKeysToWindow(sessionName, windowName, keys string) error {
	return run.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), keys, "Enter").
		Timeout(tmuxTimeout).
		Run()
}

func SendText(target, text string) error {
	err := run.Command("tmux", "send-keys", "-t", target, "-l", text).
		Timeout(tmuxTimeout).
		Run()
	if err != nil {
		return err
	}
	return run.Command("tmux", "send-keys", "-t", target, "Enter").
		Timeout(tmuxTimeout).
		Run()
}

func SendKeys(target string, keys ...string) error {
	args := append([]string{"send-keys", "-t", target}, keys...)
	return run.Command("tmux", args...).
		Timeout(tmuxTimeout).
		Run()
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
	return run.Command("tmux", "kill-session", "-t", sessionName).
		Timeout(tmuxTimeout).
		Run()
}

func ListPikoSessions() ([]string, error) {
	output, err := run.Command("tmux", "list-sessions", "-F", "#{session_name}").
		Timeout(tmuxTimeout).
		Output()
	if err != nil {
		return nil, nil
	}

	var sessions []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
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

	if cfg.DockerProject != "" {
		logsCmd := fmt.Sprintf("docker compose -p %s logs -f", cfg.DockerProject)
		NewWindow(cfg.SessionName, "logs", cfg.WorkDir, logsCmd)
	}

	return nil
}
