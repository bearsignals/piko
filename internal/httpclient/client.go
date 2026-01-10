package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const DefaultServerURL = "http://localhost:19876"

type Client struct {
	baseURL string
	http    *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

func New(opts ...ClientOption) *Client {
	c := &Client{
		baseURL: DefaultServerURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func Quick(opts ...ClientOption) *Client {
	opts = append([]ClientOption{WithTimeout(5 * time.Second)}, opts...)
	return New(opts...)
}

func Standard(opts ...ClientOption) *Client {
	opts = append([]ClientOption{WithTimeout(30 * time.Second)}, opts...)
	return New(opts...)
}

func Long(opts ...ClientOption) *Client {
	opts = append([]ClientOption{WithTimeout(60 * time.Second)}, opts...)
	return New(opts...)
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) buildURL(path string, params url.Values) string {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return u
}

func (c *Client) Get(path string, params url.Values) (*http.Response, error) {
	resp, err := c.http.Get(c.buildURL(path, params))
	if err != nil {
		return nil, wrapError(err)
	}
	return resp, nil
}

func (c *Client) Post(path string, body any, params url.Values) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	resp, err := c.http.Post(c.buildURL(path, params), "application/json", reader)
	if err != nil {
		return nil, wrapError(err)
	}
	return resp, nil
}

func (c *Client) Delete(path string, params url.Values) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.buildURL(path, params), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, wrapError(err)
	}
	return resp, nil
}

func (c *Client) IsServerRunning() bool {
	resp, err := c.Get("/api/projects", nil)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type ErrServerUnavailable struct {
	Cause error
}

func (e *ErrServerUnavailable) Error() string {
	return "piko server is not running (start with: piko server)"
}

func (e *ErrServerUnavailable) Unwrap() error {
	return e.Cause
}

func IsServerUnavailable(err error) bool {
	_, ok := err.(*ErrServerUnavailable)
	return ok
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return &ErrServerUnavailable{Cause: err}
	}

	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Op == "dial" {
			return &ErrServerUnavailable{Cause: err}
		}
	}

	return err
}
