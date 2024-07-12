package runner

import (
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/metrics"
	"log"
	"os"
	"sync"
	"time"
)

type VMWrapper struct {
	vm           *goja.Runtime
	loadTestFunc goja.Callable
	clientObject goja.Value
}

func RunStage(
	client *Client,
	concurrentUsers, duration int,
	scriptFile string,
) {
	var wg sync.WaitGroup
	vmPool := make([]*VMWrapper, concurrentUsers)

	for i := 0; i < concurrentUsers; i++ {
		vmWrapper, err := initializeVM(client, scriptFile)
		if err != nil {
			log.Fatalf("Error initializing VM: %v", err)
		}
		vmPool[i] = vmWrapper
	}

	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(vmw *VMWrapper) {
			defer wg.Done()
			vmw.runLoadTest(time.Duration(duration) * time.Second)
		}(vmPool[i])
	}

	wg.Wait()
	log.Println("Stage completed.")
}

func initializeVM(
	client *Client,
	scriptFile string,
) (*VMWrapper, error) {
	vm := goja.New()

	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		log.Println(call.Arguments)
		return goja.Undefined()
	})
	_ = vm.Set("console", console)

	exports := vm.NewObject()
	_ = vm.Set("exports", exports)

	if err := RegisterClientMethods(vm, client); err != nil {
		return nil, err
	}

	script, err := os.ReadFile(scriptFile)
	if err != nil {
		return nil, err
	}
	if _, err := vm.RunString(string(script)); err != nil {
		return nil, err
	}

	loadTestFunc, ok := goja.AssertFunction(exports.Get("loadTest"))
	if !ok {
		return nil, err
	}

	clientObject := vm.GlobalObject().Get("client")

	return &VMWrapper{
		vm:           vm,
		loadTestFunc: loadTestFunc,
		clientObject: clientObject,
	}, nil
}

func (vmw *VMWrapper) runLoadTest(duration time.Duration) {
	end := time.Now().Add(duration)
	for time.Now().Before(end) {
		start := time.Now()
		_, err := vmw.loadTestFunc(goja.Undefined(), vmw.clientObject)
		elapsed := time.Since(start)
		success := err == nil
		metrics.AddRequest(elapsed, success)
		if err != nil {
			log.Printf("Error running load test function: %v", err)
		}
		time.Sleep(time.Second)
	}
}
