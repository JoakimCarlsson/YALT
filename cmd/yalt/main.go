package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/metrics"
	"github.com/joakimcarlsson/yalt/internal/runner"
	"log"
	"os"
	"time"
)

type Options struct {
	Thresholds map[string][]string `json:"thresholds"`
	Stages     []Stage             `json:"stages"`
}

type Stage struct {
	Duration string `json:"duration"`
	Target   int    `json:"target"`
}

func main() {
	scriptFile := "C:\\Users\\JCarlsson\\Documents\\Test\\test.js"
	flag.Parse()

	log.Println("Loading JavaScript file...")

	options, err := extractOptions(scriptFile)
	if err != nil {
		log.Fatalf("failed to extract options: %v", err)
	}

	log.Printf("Options: %+v\n", options)
	log.Printf("Thresholds: %+v\n", options.Thresholds)

	client := runner.NewClient()
	go metrics.TrackMetrics()

	for _, stage := range options.Stages {
		duration, _ := time.ParseDuration(stage.Duration)
		log.Printf("Starting stage: %s with %d concurrent users for %d seconds", stage.Duration, stage.Target, int(duration.Seconds()))
		runner.RunStage(client, stage.Target, int(duration.Seconds()), scriptFile)
	}

	log.Println("Load test completed. Checking thresholds...")
	metrics.CheckThresholds(options.Thresholds)
	metrics.Summary()
	log.Println("Load test finished.")
}

func extractOptions(jsFile string) (*Options, error) {
	vm := goja.New()

	exports := vm.NewObject()
	_ = vm.Set("exports", exports)

	script, err := os.ReadFile(jsFile)
	if err != nil {
		return nil, err
	}

	_, err = vm.RunString(string(script))
	if err != nil {
		return nil, err
	}

	optionsVal := exports.Get("options")
	if goja.IsUndefined(optionsVal) {
		log.Println("options is undefined in the script")
		return nil, fmt.Errorf("options not found in script")
	}

	optionsJSON, err := json.Marshal(optionsVal)
	if err != nil {
		log.Println("failed to marshal options to JSON:", err)
		return nil, err
	}

	log.Println("Options JSON:", string(optionsJSON))

	var options Options
	err = json.Unmarshal(optionsJSON, &options)
	if err != nil {
		log.Println("failed to unmarshal options JSON:", err)
		return nil, err
	}

	return &options, nil
}
