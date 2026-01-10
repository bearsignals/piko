package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gwuah/piko/internal/httpclient"
	"github.com/spf13/cobra"
)

const RequiresAnnotation = "requires"

const (
	ToolGit    = "git"
	ToolDocker = "docker"
	ToolTmux   = "tmux"
	ToolFzf    = "fzf"
)

var toolNames = map[string]string{
	ToolGit:    "Git",
	ToolDocker: "Docker",
	ToolTmux:   "tmux",
	ToolFzf:    "fzf",
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check if required tools are installed",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	fmt.Println("Required Tools:")
	tools := []struct {
		id       string
		required bool
	}{
		{ToolGit, true},
		{ToolDocker, false},
		{ToolTmux, true},
		{ToolFzf, false},
	}

	var missingRequired bool
	for _, tool := range tools {
		name := toolNames[tool.id]
		ok, version := CheckTool(tool.id)
		if ok {
			fmt.Printf("  %s✓%s %s: %s\n", colorGreen, colorReset, name, version)
		} else if tool.required {
			fmt.Printf("  %s✗%s %s: not found\n", colorRed, colorReset, name)
			missingRequired = true
		} else {
			fmt.Printf("  - %s: not found (optional)\n", name)
		}
	}

	fmt.Println()
	fmt.Println("Services:")

	client := httpclient.Quick()
	if client.IsServerRunning() {
		fmt.Printf("  %s✓%s piko server: running at %s\n", colorGreen, colorReset, client.BaseURL())
	} else {
		fmt.Printf("  %s✗%s piko server: not running (start with: piko server)\n", colorRed, colorReset)
	}

	fmt.Println()
	fmt.Println("Hooks:")
	checkClaudeHooksForEnvironments()
	checkHookLog()

	if missingRequired {
		return fmt.Errorf("missing required tools")
	}

	return nil
}

const hookLogPath = "/tmp/piko-hook.log"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
)

func checkClaudeHooksForEnvironments() {
	ctx, err := NewContextWithoutProject()
	if err != nil {
		fmt.Printf("  ✗ failed to open database: %v\n", err)
		return
	}
	defer ctx.Close()

	projects, err := ctx.DB.ListProjects()
	if err != nil {
		fmt.Printf("  ✗ failed to list projects: %v\n", err)
		return
	}

	if len(projects) == 0 {
		fmt.Println("  no projects registered")
		return
	}

	for _, project := range projects {
		environments, err := ctx.DB.ListEnvironmentsByProject(project.ID)
		if err != nil {
			fmt.Printf("  ✗ %s: failed to list environments\n", project.Name)
			continue
		}

		for _, env := range environments {
			checkClaudeHooksForEnv(project.Name, env.Name, env.Path)
		}
	}
	fmt.Println()
}

func checkClaudeHooksForEnv(projectName, envName, envPath string) {
	fullName := projectName + "/" + envName
	settingsPath := filepath.Join(envPath, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, fullName)
		return
	}
	if err != nil {
		fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, fullName)
		return
	}

	var settings struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, fullName)
		return
	}

	var missing []string
	for _, hook := range RequiredCCHooks {
		if _, ok := settings.Hooks[hook]; !ok {
			missing = append(missing, hook)
		}
	}

	if len(missing) == 0 {
		fmt.Printf("  %s✓%s %s\n", colorGreen, colorReset, fullName)
	} else if len(missing) < len(RequiredCCHooks) {
		fmt.Printf("  %s-%s %s %smissing: %s%s\n", colorYellow, colorReset, fullName, colorDim, strings.Join(missing, ", "), colorReset)
	} else {
		fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, fullName)
	}
}

func checkHookLog() {
	info, err := os.Stat(hookLogPath)
	if os.IsNotExist(err) {
		fmt.Printf("  Runtime Logs: %s(tail -f %s)%s no activity yet\n", colorDim, hookLogPath, colorReset)
		return
	}
	if err != nil {
		fmt.Printf("  Runtime Logs: %s(tail -f %s)%s error reading log\n", colorDim, hookLogPath, colorReset)
		return
	}

	age := time.Since(info.ModTime())
	fmt.Printf("  Runtime Logs: %s(tail -f %s)%s last update %s ago\n", colorDim, hookLogPath, colorReset, formatDuration(age))
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func CheckTool(binary string) (bool, string) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return false, ""
	}

	var versionArg string
	switch binary {
	case ToolGit:
		versionArg = "--version"
	case ToolDocker:
		versionArg = "--version"
	case ToolTmux:
		versionArg = "-V"
	case ToolFzf:
		versionArg = "--version"
	default:
		return true, path
	}

	out, err := exec.Command(binary, versionArg).Output()
	if err != nil {
		return true, path
	}

	version := strings.TrimSpace(string(out))
	version = strings.Split(version, "\n")[0]
	return true, version
}

func ValidateRequiredTools(cmd *cobra.Command) error {
	requires, ok := cmd.Annotations[RequiresAnnotation]
	if !ok || requires == "" {
		return nil
	}

	tools := strings.Split(requires, ",")
	var missing []string

	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if ok, _ := CheckTool(tool); !ok {
			name := toolNames[tool]
			if name == "" {
				name = tool
			}
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s (run 'piko doctor' for details)", strings.Join(missing, ", "))
	}

	return nil
}

func Requires(tools ...string) map[string]string {
	return map[string]string{
		RequiresAnnotation: strings.Join(tools, ","),
	}
}
