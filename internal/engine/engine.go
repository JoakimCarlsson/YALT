package engine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/joakimcarlsson/yalt/internal/config"
	"github.com/joakimcarlsson/yalt/internal/http"
	"github.com/joakimcarlsson/yalt/internal/metrics"
	"github.com/joakimcarlsson/yalt/internal/models"
	"github.com/joakimcarlsson/yalt/internal/virtualuser"
)

const progressBarLength = 30

type Engine struct {
	pool    *virtualuser.UserPool
	options *models.Options
	metrics *metrics.Metrics
}

// Run starts the engine
func (e *Engine) Run() error {
	for i, stage := range e.options.Stages {
		if err := e.runStage(stage, i+1); err != nil {
			return fmt.Errorf("error running stage: %w", err)
		}
		log.Println("Stage completed")
	}
	e.metrics.CalculateAndDisplayMetrics()
	return nil
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

// runStage runs a stage with a given target number of virtual users
func (e *Engine) runStage(stage models.Stage, stageNumber int) error {
	log.Printf("Running stage with target %d for %s\n", stage.Target, stage.Duration)

	ctx, cancel, taskChan, ticker, err := e.initializeStage(stage)
	if err != nil {
		return err
	}
	defer cancel()
	defer ticker.Stop()

	var wg sync.WaitGroup

	e.startVirtualUsers(&wg, stage.Target, taskChan, ctx)

	go e.displayStageProgress(stage, stageNumber)

	e.dispatchTasks(ctx, taskChan, ticker, stage.Target)

	wg.Wait()
	return nil
}

// initializeStage initializes the stage with context, channel, and ticker
func (e *Engine) initializeStage(stage models.Stage) (
	context.Context,
	context.CancelFunc,
	chan struct{},
	*time.Ticker,
	error,
) {
	duration, err := time.ParseDuration(stage.Duration)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("invalid duration format: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	taskChan := make(chan struct{}, stage.Target)
	ticker := time.NewTicker(time.Second / time.Duration(stage.Target))

	return ctx, cancel, taskChan, ticker, nil
}

// startVirtualUsers starts the virtual user goroutines
func (e *Engine) startVirtualUsers(
	wg *sync.WaitGroup,
	target int,
	taskChan chan struct{},
	ctx context.Context,
) {
	for i := 0; i < target; i++ {
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
}

// dispatchTasks dispatches tasks to virtual users at a controlled rate
func (e *Engine) dispatchTasks(
	ctx context.Context,
	taskChan chan struct{},
	ticker *time.Ticker,
	target int,
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(taskChan)
				return
			case <-ticker.C:
				for i := 0; i < target; i++ {
					select {
					case taskChan <- struct{}{}:
					case <-ctx.Done():
						close(taskChan)
						return
					}
				}
			}
		}
	}()
}

// displayStageProgress displays the stage progress with animations
func (e *Engine) displayStageProgress(
	stage models.Stage,
	stageNumber int,
) {
	duration, _ := time.ParseDuration(stage.Duration)
	startTime := time.Now()

	for {
		elapsed := time.Since(startTime)
		if elapsed >= duration {
			fmt.Printf("\rRunning stage %d [%s] %s / %s\n", stageNumber, strings.Repeat("=", progressBarLength), stage.Duration, stage.Duration)
			return
		}

		progress := float64(elapsed) / float64(duration)
		bar := int(progress * progressBarLength)
		fmt.Printf("\rRunning stage %d [%s%s] %ds / %s", stageNumber, strings.Repeat("=", bar), strings.Repeat("-", progressBarLength-bar), int(elapsed.Seconds()), stage.Duration)

		time.Sleep(time.Second)
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
