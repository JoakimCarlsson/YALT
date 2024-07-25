package config

import (
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/joakimcarlsson/yalt/internal/models"
	"log"
	"os"
	"time"
)

func LoadConfig(scriptPath string) (*models.Options, []byte, error) {
	vm := goja.New()

	exports := vm.NewObject()
	_ = vm.Set("exports", exports)

	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading script file: %w", err)
	}

	if _, err = vm.RunString(string(script)); err != nil {
		return nil, nil, fmt.Errorf("error running script: %w", err)
	}

	optionsVal := exports.Get("options")
	if goja.IsUndefined(optionsVal) {
		log.Println("options is undefined in the script")
		return nil, nil, fmt.Errorf("options not found in script")
	}

	optionsJSON, err := json.Marshal(optionsVal)
	if err != nil {
		log.Println("failed to marshal options to JSON:", err)
		return nil, nil, fmt.Errorf("error marshaling options: %w", err)
	}

	var options models.Options
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		log.Println("failed to unmarshal options JSON:", err)
		return nil, nil, fmt.Errorf("error unmarshaling options: %w", err)
	}

	if err := validateOptions(&options); err != nil {
		return nil, nil, fmt.Errorf("invalid options: %w", err)
	}

	return &options, script, nil
}

func validateOptions(options *models.Options) error {
	if options == nil {
		return fmt.Errorf("options cannot be nil")
	}
	if len(options.Stages) == 0 {
		return fmt.Errorf("at least one stage is required")
	}
	for _, stage := range options.Stages {
		if stage.Target <= 0 {
			return fmt.Errorf("stage target must be greater than 0")
		}
		if _, err := time.ParseDuration(stage.Duration); err != nil {
			return fmt.Errorf("invalid stage duration: %w", err)
		}
		if stage.RampUp != "" {
			if _, err := time.ParseDuration(stage.RampUp); err != nil {
				return fmt.Errorf("invalid stage ramp-up duration: %w", err)
			}
		}
		if stage.RampDown != "" {
			if _, err := time.ParseDuration(stage.RampDown); err != nil {
				return fmt.Errorf("invalid stage ramp-down duration: %w", err)
			}
		}
	}
	return nil
}
