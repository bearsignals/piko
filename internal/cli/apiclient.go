package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultServerURL = "http://localhost:19876"

type APIClient struct {
	baseURL string
	client  *http.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{
		baseURL: defaultServerURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *APIClient) IsServerRunning() bool {
	resp, err := c.client.Get(c.baseURL + "/api/projects")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type apiResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (c *APIClient) CreateEnvironment(projectID int64, name, branch string) error {
	body, _ := json.Marshal(map[string]string{"name": name, "branch": branch})
	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/projects/%d/environments", c.baseURL, projectID),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) DestroyEnvironment(projectID int64, name string, removeVolumes bool) error {
	url := fmt.Sprintf("%s/api/projects/%d/environments/%s", c.baseURL, projectID, name)
	if removeVolumes {
		url += "?volumes=true"
	}
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) Up(projectID int64, name string) error {
	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/projects/%d/environments/%s/up", c.baseURL, projectID, name),
		"application/json",
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) Down(projectID int64, name string) error {
	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/projects/%d/environments/%s/down", c.baseURL, projectID, name),
		"application/json",
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) Restart(projectID int64, name, service string) error {
	url := fmt.Sprintf("%s/api/projects/%d/environments/%s/restart", c.baseURL, projectID, name)
	if service != "" {
		url += "?service=" + service
	}
	resp, err := c.client.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) parseResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("server error: %s", string(body))
		}
		return nil
	}

	if !result.Success && result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}
