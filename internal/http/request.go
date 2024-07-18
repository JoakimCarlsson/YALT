package http

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
)

// Fetch performs an HTTP request based on the provided configuration
// Fetch performs an HTTP request based on the provided configuration
func (c *Client) Fetch(config map[string]interface{}) error {
	// Extract and validate method
	method, ok := config["method"].(string)
	if !ok {
		method = "GET"
	}

	// Extract and validate URL
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return errors.New("url is required and must be a string")
	}

	// Extract body if present
	var body io.Reader
	if bodyStr, ok := config["body"].(string); ok {
		body = bytes.NewBufferString(bodyStr)
	} else {
		body = nil
	}

	// Create new HTTP request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Println("Failed to create request:", err)
		return err
	}

	// Set headers if present
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
		log.Println("Request failed with error:", err)
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return err
	}

	return nil
}
