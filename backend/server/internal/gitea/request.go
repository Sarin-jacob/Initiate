package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// doRequest handles building the HTTP request, injecting headers, and returning standard errors
func (c *Client) doRequest(ctx context.Context, method, path string, payload interface{}) (*http.Response, error) {
	endpoint := fmt.Sprintf("%s/api/v1%s", c.BaseURL, path)

	var reqBody io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request payload: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 1. Inject API Token and Content Type
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.AdminToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 2. Execute Request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	// 3. Handle standard HTTP error codes
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("gitea API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}