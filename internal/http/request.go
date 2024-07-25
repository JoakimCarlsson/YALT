package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Fetch performs an HTTP request based on the provided configuration
func (c *Client) Fetch(config map[string]interface{}) (map[string]interface{}, error) {
	method, ok := config["method"].(string)
	if !ok {
		method = "GET"
	}

	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, errors.New("url is required and must be a string")
	}

	var body io.Reader
	if bodyStr, ok := config["body"].(string); ok {
		body = bytes.NewBufferString(bodyStr)
	} else {
		body = nil
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if headerValue, ok := value.(string); ok {
				req.Header.Set(key, headerValue)
			} else {
				log.Println("Invalid header value for key:", key)
			}
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	responseDetails := map[string]interface{}{
		"statusCode":    resp.StatusCode,
		"statusMessage": resp.Status,
		"headers":       resp.Header,
		"body":          string(responseBody),
	}

	return responseDetails, nil
}
