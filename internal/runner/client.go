package runner

import (
	"fmt"
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

func (c *Client) Request(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Failed to create request:", err)
		return nil
	}
	return req
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		log.Println("Request is nil, cannot send request")
		return nil, fmt.Errorf("request is nil")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		log.Println("Request failed with error:", err)
		return nil, err
	}
	return resp, nil
}

func RegisterClientMethods(vm *goja.Runtime, client *Client) {
	clientObj := vm.NewObject()
	clientObj.Set("Request", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		return vm.ToValue(client.Request(url))
	})
	clientObj.Set("Do", func(call goja.FunctionCall) goja.Value {
		req := call.Argument(0).Export().(*http.Request)
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error performing request:", err)
			return goja.Null()
		}
		return vm.ToValue(resp)
	})
	vm.Set("client", clientObj)
}
