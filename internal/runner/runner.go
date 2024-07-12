package runner

import (
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/metrics"
	"log"
	"os"
	"sync"
	"time"
)

func RunStage(client *Client, concurrentUsers, duration int, scriptFile string) {
	var wg sync.WaitGroup
	vmPool := make(chan *goja.Runtime, concurrentUsers)

	for i := 0; i < concurrentUsers; i++ {
		vm := goja.New()

		console := vm.NewObject()
		console.Set("log", func(call goja.FunctionCall) goja.Value {
			log.Println(call.Arguments)
			return goja.Undefined()
		})
		vm.Set("console", console)

		exports := vm.NewObject()
		vm.Set("exports", exports)

		err := RegisterClientMethods(vm, client)
		if err != nil {
			log.Printf("Error registering client methods: %v", err)
			continue
		}

		script, err := os.ReadFile(scriptFile)
		if err != nil {
			log.Printf("Error reading script file: %v", err)
			continue
		}

		_, err = vm.RunString(string(script))
		if err != nil {
			log.Printf("Error running script: %v", err)
			continue
		}

		vmPool <- vm
	}

	requestsPerSecond := concurrentUsers
	requestInterval := time.Second / time.Duration(requestsPerSecond)
	endTime := time.Now().Add(time.Duration(duration) * time.Second)

	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm := <-vmPool
			defer func() { vmPool <- vm }()

			exports := vm.Get("exports").ToObject(vm)
			loadTestFunc, ok := goja.AssertFunction(exports.Get("loadTest"))
			if !ok {
				log.Println("loadTest function not found in script")
				return
			}

			for time.Now().Before(endTime) {
				start := time.Now()
				_, err := loadTestFunc(goja.Undefined(), vm.Get("client"))
				duration := time.Since(start)
				success := err == nil
				metrics.AddRequest(duration, success)
				if err != nil {
					log.Printf("Error running load test function: %v", err)
				}

				elapsed := time.Since(start)
				sleepDuration := requestInterval - elapsed
				if sleepDuration > 0 {
					time.Sleep(sleepDuration)
				}
			}
		}()
	}

	wg.Wait()
	log.Println("Stage completed.")
}
