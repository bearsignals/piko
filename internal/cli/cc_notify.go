package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var ccNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Send a notification from Claude Code hook",
	Long:  `Reads Claude Code hook JSON from stdin and sends a notification to the Piko server.`,
	RunE:  runCCNotify,
}

var ccNotifyServerURL string

func init() {
	ccCmd.AddCommand(ccNotifyCmd)
	ccNotifyCmd.Flags().StringVar(&ccNotifyServerURL, "server", "http://localhost:19876", "Piko server URL")
}

type hookInput struct {
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
}

type notifyRequest struct {
	ProjectID        int64  `json:"project_id"`
	ProjectName      string `json:"project_name"`
	EnvName          string `json:"env_name"`
	TmuxSession      string `json:"tmux_session"`
	TmuxTarget       string `json:"tmux_target"`
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
}

func runCCNotify(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var hook hookInput
	if len(input) > 0 {
		json.Unmarshal(input, &hook)
	}

	projectID, projectName, envName, tmuxSession := detectEnvironment()
	tmuxTarget := os.Getenv("TMUX_PANE")

	req := notifyRequest{
		ProjectID:        projectID,
		ProjectName:      projectName,
		EnvName:          envName,
		TmuxSession:      tmuxSession,
		TmuxTarget:       tmuxTarget,
		NotificationType: hook.NotificationType,
		Message:          hook.Message,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(ccNotifyServerURL+"/api/orchestra/notify", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	return nil
}

func detectEnvironment() (projectID int64, projectName, envName, tmuxSession string) {
	if id := os.Getenv("PIKO_PROJECT_ID"); id != "" {
		projectID, _ = strconv.ParseInt(id, 10, 64)
	}

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

	if projectID == 0 && projectName != "" {
		ctx, err := NewContextWithoutProject()
		if err == nil {
			defer ctx.Close()
			project, err := ctx.DB.GetProjectByName(projectName)
			if err == nil && project != nil {
				projectID = project.ID
			}
		}
	}

	return
}
