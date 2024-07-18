package http

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
)

func (c *Client) Fetch(config map[string]interface{}) error {
	method := "GET"
	if config["method"] != nil {
		method = config["method"].(string)
	}

	url := config["url"].(string)
	if url == "" {
		return errors.New("url is required")
	}

	var body []byte
	if config["body"] != nil {
		body = []byte(config["body"].(string))
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Println("Failed to create request:", err)
		return err
	}

	if config["headers"] != nil {
		headers := config["headers"].(map[string]interface{})
		for key, value := range headers {
			req.Header.Set(key, value.(string))
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
			return
		}
	}(resp.Body)

	return nil
}
