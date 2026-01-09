package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var respondCmd = &cobra.Command{
	Use:   "respond [notification-id] [response]",
	Short: "Respond to a Claude Code notification",
	Long:  `Respond to a pending Claude Code notification. If no arguments provided, shows pending notifications.`,
	Args:  cobra.MaximumNArgs(2),
	RunE:  runRespond,
}

var respondServerURL string

func init() {
	rootCmd.AddCommand(respondCmd)
	respondCmd.Flags().StringVar(&respondServerURL, "server", "http://localhost:19876", "Piko server URL")
}

type ccNotification struct {
	ID               string `json:"id"`
	ProjectID        int64  `json:"project_id"`
	ProjectName      string `json:"project_name"`
	EnvName          string `json:"env_name"`
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
}

type respondRequest struct {
	NotificationID string `json:"notification_id"`
	Response       string `json:"response"`
}

func runRespond(cmd *cobra.Command, args []string) error {
	notifications, err := fetchNotifications()
	if err != nil {
		return fmt.Errorf("failed to fetch notifications: %w", err)
	}

	if len(notifications) == 0 {
		fmt.Println("No pending notifications")
		return nil
	}

	var notificationID, response string

	if len(args) == 0 {
		fmt.Println("Pending notifications:")
		fmt.Println()
		for i, n := range notifications {
			fmt.Printf("  [%d] %s/%s (%s)\n", i+1, n.ProjectName, n.EnvName, n.NotificationType)
			fmt.Printf("      %s\n", n.Message)
			fmt.Printf("      ID: %s\n", n.ID)
			fmt.Println()
		}
		return nil
	}

	if len(args) >= 1 {
		notificationID = args[0]
	}

	if len(args) >= 2 {
		response = args[1]
	} else {
		fmt.Print("Response: ")
		var input string
		fmt.Scanln(&input)
		response = strings.TrimSpace(input)
	}

	return sendResponse(notificationID, response)
}

func fetchNotifications() ([]ccNotification, error) {
	resp, err := http.Get(respondServerURL + "/api/orchestra/notifications")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var notifications []ccNotification
	if err := json.Unmarshal(body, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

func sendResponse(notificationID, response string) error {
	req := respondRequest{
		NotificationID: notificationID,
		Response:       response,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(respondServerURL+"/api/orchestra/respond", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("Response sent successfully")
	return nil
}
