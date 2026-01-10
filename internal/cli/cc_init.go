package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ccInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Claude Code hooks for the current environment",
	Long:  `Creates or updates .claude/settings.json with hooks that integrate with Piko Orchestra.`,
	RunE:  runCCInit,
}

func init() {
	ccCmd.AddCommand(ccInitCmd)
}

type claudeSettings struct {
	Hooks map[string][]hookMatcher `json:"hooks"`
}

type hookMatcher struct {
	Matcher string       `json:"matcher,omitempty"`
	Hooks   []hookConfig `json:"hooks"`
}

type hookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func runCCInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	claudeDir := filepath.Join(cwd, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	var settings claudeSettings
	if data, err := os.ReadFile(settingsPath); err == nil {
		json.Unmarshal(data, &settings)
	}

	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]hookMatcher)
	}

	settings.Hooks["PermissionRequest"] = []hookMatcher{
		{
			Hooks: []hookConfig{
				{
					Type:    "command",
					Command: "piko cc notify",
					Timeout: 10,
				},
			},
		},
	}

	settings.Hooks["Notification"] = []hookMatcher{
		{
			Hooks: []hookConfig{
				{
					Type:    "command",
					Command: "piko cc notify",
					Timeout: 10,
				},
			},
		},
	}

	// settings.Hooks["Stop"] = []hookMatcher{
	// 	{
	// 		Hooks: []hookConfig{
	// 			{
	// 				Type:    "command",
	// 				Command: "piko cc notify",
	// 				Timeout: 10,
	// 			},
	// 		},
	// 	},
	// }

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("Created %s with Orchestra hooks\n", settingsPath)
	fmt.Println()
	fmt.Println("Claude Code will now send notifications to Piko when it needs input.")
	fmt.Println("Make sure the Piko server is running: piko serve")

	return nil
}
