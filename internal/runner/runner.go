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
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
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
				return
			}

			script, err := os.ReadFile(scriptFile)
			if err != nil {
				log.Printf("Error reading script file: %v", err)
				return
			}
			_, err = vm.RunString(string(script))
			if err != nil {
				log.Printf("Error running script: %v", err)
				return
			}
			loadTestFunc, ok := goja.AssertFunction(exports.Get("loadTest"))
			if !ok {
				log.Println("loadTest function not found in script")
				return
			}
			end := time.Now().Add(time.Duration(duration) * time.Second)
			for time.Now().Before(end) {
				start := time.Now()
				_, err := loadTestFunc(goja.Undefined(), vm.GlobalObject().Get("client"))
				duration := time.Since(start)
				success := err == nil
				metrics.AddRequest(duration, success)
				if err != nil {
					log.Printf("Error running load test function: %v", err)
				}
				time.Sleep(time.Second)
			}
		}()
	}
	wg.Wait()
	log.Println("Stage completed.")
}
