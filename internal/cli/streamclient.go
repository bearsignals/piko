package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/gwuah/piko/internal/httpclient"
)

type StreamClient struct {
	baseURL string
}

func NewStreamClient() *StreamClient {
	return &StreamClient{
		baseURL: httpclient.DefaultServerURL,
	}
}

type LogMessage struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Stream string `json:"stream"`
	Data   string `json:"data"`
}

type CompleteMessage struct {
	Type        string       `json:"type"`
	Success     bool         `json:"success"`
	Environment *Environment `json:"environment,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type Environment struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Status string `json:"status"`
}

type CreateRequest struct {
	Action      string `json:"action"`
	Environment string `json:"environment"`
	Branch      string `json:"branch"`
}

func (c *StreamClient) CreateEnvironmentStream(projectID int64, name, branch string) error {
	wsURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = fmt.Sprintf("/api/projects/%d/environments/create/stream", projectID)

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	req := CreateRequest{
		Action:      "create",
		Environment: name,
		Branch:      branch,
	}
	if err := conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("connection error: %w", err)
		}

		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message, &baseMsg); err != nil {
			continue
		}

		switch baseMsg.Type {
		case "log":
			var logMsg LogMessage
			if err := json.Unmarshal(message, &logMsg); err != nil {
				continue
			}
			c.printLog(logMsg)

		case "complete":
			var completeMsg CompleteMessage
			if err := json.Unmarshal(message, &completeMsg); err != nil {
				return fmt.Errorf("failed to parse completion: %w", err)
			}

			if !completeMsg.Success {
				return fmt.Errorf("%s", completeMsg.Error)
			}
			return nil
		}
	}
}

func (c *StreamClient) printLog(msg LogMessage) {
	prefix := c.sourcePrefix(msg.Source)

	data := strings.TrimSuffix(msg.Data, "\n")
	if data == "" {
		return
	}

	for _, line := range strings.Split(data, "\n") {
		fmt.Printf("[%s] %s\n", prefix, line)
	}
}

func (c *StreamClient) sourcePrefix(source string) string {
	switch source {
	case "git":
		return "git"
	case "docker":
		return "docker"
	case "script:prepare":
		return "prepare"
	case "script:setup":
		return "setup"
	case "piko":
		return "piko"
	default:
		return source
	}
}
