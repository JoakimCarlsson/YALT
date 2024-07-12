package runner

import (
	"bytes"
	"errors"
	"log"
	"net/http"

	"github.com/dop251/goja"
)

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{client: &http.Client{}}
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

	_, err = c.client.Do(req)
	if err != nil {
		log.Println("Request failed with error:", err)
		return err
	}

	return nil
}

func RegisterClientMethods(vm *goja.Runtime, client *Client) error {
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
