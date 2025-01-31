package http

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/metrics"
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
func NewClient(metrics *metrics.Metrics) *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   1000,
		MaxConnsPerHost:       100,
	}

	client := &http.Client{
		Transport: metrics.NewMetricsRoundTripper(transport, metrics),
		Timeout:   30 * time.Second,
	}
	return &Client{client: client}
}

// RegisterClientMethods registers the fetch method of the Client in the Goja runtime
func RegisterClientMethods(
	vm *goja.Runtime,
	client *Client,
) error {
	clientObj := vm.NewObject()
	if err := clientObj.Set("fetch", func(call goja.FunctionCall) goja.Value {
		config, ok := call.Argument(0).Export().(map[string]interface{})
		if !ok {
			log.Println("Invalid argument type, expected map[string]interface{}")
			return vm.ToValue("Invalid argument type")
		}

		responseDetails, err := client.Fetch(config)
		if err != nil {
			log.Println("Error performing request:", err)
			return vm.ToValue(map[string]interface{}{
				"error": err.Error(),
			})
		}

		return vm.ToValue(responseDetails)
	}); err != nil {
		return fmt.Errorf("error setting fetch method: %w", err)
	}

	if err := vm.Set("client", clientObj); err != nil {
		return fmt.Errorf("error setting client object: %w", err)
	}

	return nil
}
