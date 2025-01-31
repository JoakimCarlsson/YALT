package virtualuser

import (
	"context"
	"fmt"
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/http"
	"log"
)

// VirtualUser represents a virtual user.
type VirtualUser struct {
	loadTestFunc goja.Callable
	clientObject goja.Value
}

// Run runs the virtual user for the specified duration, sending requests.
func (vu *VirtualUser) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		_, err := vu.loadTestFunc(goja.Undefined(), vu.clientObject)
		if err != nil {
			log.Printf("Error running load test function: %v", err)
			return fmt.Errorf("error running load test function: %w", err)
		}
		return nil
	}
}

// CreateVu creates a new VirtualUser.
func CreateVu(
	client *http.Client,
	scriptContent []byte,
) (*VirtualUser, error) {
	runtime, err := setupRuntime(client)
	if err != nil {
		return nil, fmt.Errorf("failed to set up runtime: %w", err)
	}

	if _, err := runtime.RunString(string(scriptContent)); err != nil {
		return nil, fmt.Errorf("failed to run script: %w", err)
	}

	loadTestFunc, err := getLoadTestFunc(runtime)
	if err != nil {
		return nil, fmt.Errorf("error getting load test function: %w", err)
	}

	clientObject := runtime.GlobalObject().Get("client")

	return &VirtualUser{
		loadTestFunc: loadTestFunc,
		clientObject: clientObject,
	}, nil
}

// setupRuntime initializes the JavaScript runtime and registers necessary objects and methods.
func setupRuntime(client *http.Client) (*goja.Runtime, error) {
	runtime := goja.New()

	logHandler := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			msg := call.Argument(0).String()
			log.Println(msg)
		}
		return goja.Undefined()
	}

	console := runtime.NewObject()
	if err := console.Set("log", logHandler); err != nil {
		return nil, fmt.Errorf("failed to set console.log: %w", err)
	}

	if err := runtime.Set("console", console); err != nil {
		return nil, fmt.Errorf("failed to set console object: %w", err)
	}

	exports := runtime.NewObject()
	if err := runtime.Set("exports", exports); err != nil {
		return nil, fmt.Errorf("failed to set exports object: %w", err)
	}

	if err := http.RegisterClientMethods(runtime, client); err != nil {
		return nil, fmt.Errorf("failed to register client methods: %w", err)
	}

	return runtime, nil
}

// getLoadTestFunc retrieves the loadTest function from the exports object.
func getLoadTestFunc(runtime *goja.Runtime) (goja.Callable, error) {
	exports := runtime.Get("exports")
	loadTestFunc, ok := goja.AssertFunction(exports.ToObject(runtime).Get("loadTest"))
	if !ok {
		return nil, fmt.Errorf("loadTest function not found in exports")
	}
	return loadTestFunc, nil
}
