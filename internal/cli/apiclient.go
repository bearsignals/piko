package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gwuah/piko/internal/httpclient"
)

type APIClient struct {
	client *httpclient.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{
		client: httpclient.Long(),
	}
}

func (c *APIClient) IsServerRunning() bool {
	return c.client.IsServerRunning()
}

type apiResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (c *APIClient) CreateEnvironment(projectID int64, name, branch string) error {
	resp, err := c.client.Post(
		fmt.Sprintf("/api/projects/%d/environments", projectID),
		map[string]string{"name": name, "branch": branch},
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) DestroyEnvironment(projectID int64, name string, removeVolumes bool) error {
	var params url.Values
	if !removeVolumes {
		params = url.Values{}
		params.Set("keep-volumes", "true")
	}
	resp, err := c.client.Delete(
		fmt.Sprintf("/api/projects/%d/environments/%s", projectID, name),
		params,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) Up(projectID int64, name string) error {
	resp, err := c.client.Post(
		fmt.Sprintf("/api/projects/%d/environments/%s/up", projectID, name),
		nil,
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
		fmt.Sprintf("/api/projects/%d/environments/%s/down", projectID, name),
		nil,
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.parseResponse(resp)
}

func (c *APIClient) Restart(projectID int64, name, service string) error {
	var params url.Values
	if service != "" {
		params = url.Values{}
		params.Set("service", service)
	}
	resp, err := c.client.Post(
		fmt.Sprintf("/api/projects/%d/environments/%s/restart", projectID, name),
		nil,
		params,
	)
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
