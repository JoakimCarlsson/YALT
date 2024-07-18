package engine

import (
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/models"
	"github.com/joakimcarlsson/yalt/internal/virtualuser"
	"log"
	"os"
	"sync"
	"time"
)

type Engine struct {
	pool    *virtualuser.UserPool
	options *models.Options
}

// Run starts the engine wroom. wroom
func (e Engine) Run() error {
	for _, stage := range e.options.Stages {
		e.runStage(stage)
		log.Println("Stage completed")
	}
	return nil
}

// runStage runs a stage
func (e Engine) runStage(stage models.Stage) {
	log.Printf("Running stage with target %d for %s\n", stage.Target, stage.Duration)
	duration, _ := time.ParseDuration(stage.Duration)

	var wg sync.WaitGroup

	for i := 0; i < stage.Target; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user := e.pool.Fetch()
			err := user.Run(duration)
			if err != nil {
				log.Printf("Error running virtual user: %v", err)
			}
			e.pool.Return(user)
		}()
	}

	wg.Wait()
}

// New creates a new Engine instance
func New(scriptPath string) *Engine {
	options, scriptContent, err := extractOptions(scriptPath)

	if err != nil {
		panic(err)
	}

	maxVuCount := getMaxVuCount(options)
	pool, err := virtualuser.CreatePool(maxVuCount, scriptContent)
	if err != nil {
		panic(err)
	}

	return &Engine{
		pool,
		options,
	}
}

// getMaxVuCount calculates the maximum number of virtual users
func getMaxVuCount(options *models.Options) int {
	maxVuCount := 0
	for _, stage := range options.Stages {
		if stage.Target > maxVuCount {
			maxVuCount = stage.Target
		}
	}
	return maxVuCount
}

// extractOptions extracts the options from a JavaScript file
func extractOptions(scriptPath string) (*models.Options, []byte, error) {
	vm := goja.New()

	exports := vm.NewObject()
	_ = vm.Set("exports", exports)

	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, nil, err
	}

	_, err = vm.RunString(string(script))
	if err != nil {
		return nil, nil, err
	}

	optionsVal := exports.Get("options")
	if goja.IsUndefined(optionsVal) {
		log.Println("options is undefined in the script")
		return nil, nil, fmt.Errorf("options not found in script")
	}

	optionsJSON, err := json.Marshal(optionsVal)
	if err != nil {
		log.Println("failed to marshal options to JSON:", err)
		return nil, nil, err
	}

	log.Println("Options JSON:", string(optionsJSON))

	var options models.Options
	err = json.Unmarshal(optionsJSON, &options)
	if err != nil {
		log.Println("failed to unmarshal options JSON:", err)
		return nil, nil, err
	}

	return &options, script, nil
}
