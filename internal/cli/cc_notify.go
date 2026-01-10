package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gwuah/piko/internal/httpclient"
	"github.com/gwuah/piko/internal/logger"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var ccNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Send a notification from Claude Code hook",
	Long:  `Reads Claude Code hook JSON from stdin and sends a notification to the Piko server.`,
	RunE:  runCCNotify,
}

func init() {
	ccCmd.AddCommand(ccNotifyCmd)
}

type hookInput struct {
	SessionID        string          `json:"session_id"`
	TranscriptPath   string          `json:"transcript_path"`
	Cwd              string          `json:"cwd"`
	HookEventName    string          `json:"hook_event_name"`
	Message          string          `json:"message"`
	Title            string          `json:"title"`
	NotificationType string          `json:"notification_type"`
	PermissionMode   string          `json:"permission_mode"`
	ToolName         string          `json:"tool_name"`
	ToolInput        json.RawMessage `json:"tool_input"`
	ToolUseID        string          `json:"tool_use_id"`
}

type notifyRequest struct {
	ProjectName      string `json:"project_name"`
	EnvName          string `json:"env_name"`
	TmuxSession      string `json:"tmux_session"`
	TmuxTarget       string `json:"tmux_target"`
	ParentPID        int    `json:"parent_pid"`
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
}

func runCCNotify(cmd *cobra.Command, args []string) error {
	log, err := logger.NewFileLogger("/tmp/piko-hook.log")
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer log.Close()

	log.Log("hook started")

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Log("ERROR reading stdin: %v", err)
		return fmt.Errorf("failed to read stdin: %w", err)
	}
	log.Log("stdin read: %d bytes", len(input))

	var hook hookInput
	log.Log("raw input: %s", string(input))
	if len(input) > 0 {
		if err := json.Unmarshal(input, &hook); err != nil {
			log.Log("ERROR parsing JSON: %v", err)
			return fmt.Errorf("failed to parse hook input: %w", err)
		}
	}

	log.Struct("hook_input", hook)

	projectName, envName, tmuxSession := detectEnvironment()
	tmuxPane := os.Getenv("TMUX_PANE")
	parentPID := os.Getppid()

	notificationType := hook.NotificationType
	if notificationType == "" {
		notificationType = hook.HookEventName
	}

	message := hook.Message
	if message == "" && hook.ToolName != "" {
		message = fmt.Sprintf("Permission requested for %s", hook.ToolName)
	}

	req := notifyRequest{
		ProjectName:      projectName,
		EnvName:          envName,
		TmuxSession:      tmuxSession,
		TmuxTarget:       tmuxPane,
		ParentPID:        parentPID,
		NotificationType: notificationType,
		Message:          message,
	}
	log.Struct("notifyRequest", req)

	client := httpclient.Quick()
	resp, err := client.Post("/api/orchestra/notify", req, nil)
	if err != nil {
		log.Log("ERROR http request failed: %v", err)
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log("ERROR http request failed: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Log("http response: status=%d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		log.Log("ERROR server returned non-200 status")
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Log("hook completed successfully")
	return nil
}

func detectEnvironment() (projectName, envName, tmuxSession string) {
	projectName = os.Getenv("PIKO_PROJECT")
	envName = os.Getenv("PIKO_ENV_NAME")

	if projectName != "" && envName != "" {
		tmuxSession = tmux.SessionName(projectName, envName)
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	parts := strings.Split(cwd, string(filepath.Separator))
	for i, part := range parts {
		if part == ".piko" && i+2 < len(parts) && parts[i+1] == "worktrees" {
			envName = parts[i+2]
			if i > 0 {
				projectName = parts[i-1]
			}
			break
		}
	}

	if projectName != "" && envName != "" {
		tmuxSession = tmux.SessionName(projectName, envName)
	}

	return
}
