package http

import (
	"github.com/dop251/goja"
	"log"
	"net"
	"net/http"
	"time"
)

// Client wraps an HTTP client with custom settings
type Client struct {
	client *http.Client
}

// NewClient initializes and returns a new Client with custom transport settings
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

// RegisterClientMethods registers the fetch method of the Client in the Goja runtime
func RegisterClientMethods(vm *goja.Runtime, client *Client) error {
	clientObj := vm.NewObject()
	err := clientObj.Set("fetch", func(call goja.FunctionCall) goja.Value {
		config, ok := call.Argument(0).Export().(map[string]interface{})
		if !ok {
			log.Println("Invalid argument type, expected map[string]interface{}")
			return vm.ToValue("Invalid argument type")
		}

		err := client.Fetch(config)
		if err != nil {
			log.Println("Error performing request:", err)
			return vm.ToValue("Error performing request: " + err.Error())
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
