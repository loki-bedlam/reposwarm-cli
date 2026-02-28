// Package api provides an HTTP client for the RepoSwarm API server.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to the RepoSwarm API server.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New creates an API client.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// apiResponse wraps all API responses.
type apiResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

// Get performs a GET request and unmarshals the response data.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request and unmarshals the response data.
func (c *Client) Post(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// Patch performs a PATCH request and unmarshals the response data.
func (c *Client) Patch(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPatch, path, body, result)
}

// Delete performs a DELETE request and unmarshals the response data.
func (c *Client) Delete(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodDelete, path, nil, result)
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed (401): run 'reposwarm config init' to update your token")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("not found (404): %s", path)
	}
	if resp.StatusCode >= 400 {
		var apiErr apiResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result == nil {
		return nil
	}

	// Try unwrapping { data: ... }
	var wrapped apiResponse
	if err := json.Unmarshal(respBody, &wrapped); err == nil && wrapped.Data != nil {
		return json.Unmarshal(wrapped.Data, result)
	}

	// Fall back to direct unmarshal
	return json.Unmarshal(respBody, result)
}

// Health checks the API connection.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var h HealthResponse
	if err := c.Get(ctx, "/health", &h); err != nil {
		return nil, err
	}
	return &h, nil
}
