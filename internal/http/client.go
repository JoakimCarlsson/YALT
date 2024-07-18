package http

import (
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
