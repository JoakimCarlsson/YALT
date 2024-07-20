package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/joakimcarlsson/yalt/internal/config"
	"github.com/joakimcarlsson/yalt/internal/http"
	"github.com/joakimcarlsson/yalt/internal/metrics"
	"github.com/joakimcarlsson/yalt/internal/models"
	"github.com/joakimcarlsson/yalt/internal/virtualuser"
)

type Engine struct {
	pool    *virtualuser.UserPool
	options *models.Options
	metrics *metrics.Metrics
}

// Run starts the engine
func (e *Engine) Run() error {
	for _, stage := range e.options.Stages {
		if err := e.runStage(stage); err != nil {
			return fmt.Errorf("error running stage: %w", err)
		}
		log.Println("Stage completed")
	}
	e.metrics.CalculateAndDisplayMetrics()
	return nil
}

// runStage runs a stage with a given target number of virtual users
func (e *Engine) runStage(stage models.Stage) error {
	log.Printf("Running stage with target %d for %s\n", stage.Target, stage.Duration)
	duration, err := time.ParseDuration(stage.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	taskChan := make(chan struct{}, stage.Target)

	for i := 0; i < stage.Target; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user := e.pool.Fetch()
			defer e.pool.Return(user)
			for range taskChan {
				if err := user.Run(ctx); err != nil {
					log.Printf("Error running virtual user: %v", err)
				}
			}
		}()
	}

	ticker := time.NewTicker(time.Second / time.Duration(stage.Target))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(taskChan)
			wg.Wait()
			return nil
		case <-ticker.C:
			for i := 0; i < stage.Target; i++ {
				select {
				case taskChan <- struct{}{}:
				case <-ctx.Done():
					close(taskChan)
					wg.Wait()
					return nil
				}
			}
		}
	}
}

// New creates a new Engine instance
func New(scriptPath string) (*Engine, error) {
	options, scriptContent, err := config.LoadConfig(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("error extracting options: %w", err)
	}

	maxVuCount := getMaxVuCount(options)
	metrics := metrics.NewMetrics(options.Thresholds)
	client := http.NewClient(metrics)

	pool, err := virtualuser.CreatePool(maxVuCount, scriptContent, client)
	if err != nil {
		return nil, fmt.Errorf("error creating user pool: %w", err)
	}

	return &Engine{
		pool:    pool,
		options: options,
		metrics: metrics,
	}, nil
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
