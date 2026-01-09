package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const RequiresAnnotation = "requires"

const (
	ToolGit    = "git"
	ToolDocker = "docker"
	ToolTmux   = "tmux"
)

var toolNames = map[string]string{
	ToolGit:    "Git",
	ToolDocker: "Docker",
	ToolTmux:   "tmux",
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

	tools := []struct {
		id       string
		required bool
	}{
		{ToolGit, true},
		{ToolDocker, false},
		{ToolTmux, true},
	}

	allOk := true
	for _, tool := range tools {
		name := toolNames[tool.id]
		ok, version := CheckTool(tool.id)
		if ok {
			fmt.Printf("  ✓ %s: %s\n", name, version)
		} else if tool.required {
			fmt.Printf("  ✗ %s: not found (required)\n", name)
			allOk = false
		} else {
			fmt.Printf("  - %s: not found (optional, needed for Docker mode)\n", name)
		}
	}

	fmt.Println()
	if allOk {
		fmt.Println("All required tools are installed")
	} else {
		fmt.Println("Some required tools are missing")
		return fmt.Errorf("missing required tools")
	}

	return nil
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
