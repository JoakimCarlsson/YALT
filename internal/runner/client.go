package runner

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dop251/goja"
)

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	return &Client{client: client}
}

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

func RegisterClientMethods(
	vm *goja.Runtime,
	client *Client,
) error {
	clientObj := vm.NewObject()
	err := clientObj.Set("fetch", func(call goja.FunctionCall) goja.Value {
		config := call.Argument(0).Export().(map[string]interface{})

		err := client.Fetch(config)
		if err != nil {
			log.Println("Error performing request:", err)
		}

		return goja.Undefined()
	})
	if err != nil {
		return err
	}
	err = vm.Set("client", clientObj)
	if err != nil {
		return err
	}

	return nil
}
