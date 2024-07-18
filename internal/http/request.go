package http

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
)

func (c *Client) Fetch(config map[string]interface{}) error {
	method, ok := config["method"].(string)
	if !ok {
		method = "GET"
	}

	url, ok := config["url"].(string)
	if !ok || url == "" {
		return errors.New("url is required and must be a string")
	}

	var body io.Reader
	if bodyStr, ok := config["body"].(string); ok {
		body = bytes.NewBufferString(bodyStr)
	} else {
		body = nil
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Println("Failed to create request:", err)
		return err
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
		log.Println("Request failed with error:", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Failed to close response body:", err)
		}
	}(resp.Body)

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return err
	}

	return nil
}
