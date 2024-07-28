package engine

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joakimcarlsson/yalt/internal/config"
	"github.com/joakimcarlsson/yalt/internal/http"
	"github.com/joakimcarlsson/yalt/internal/metrics"
	"github.com/joakimcarlsson/yalt/internal/models"
	"github.com/joakimcarlsson/yalt/internal/virtualuser"
)

const progressBarLength = 30

type Engine struct {
	pool        *virtualuser.UserPool
	options     *models.Options
	metrics     *metrics.Metrics
	activeUsers int64
	taskChan    chan struct{}
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
	httpMetrics := metrics.NewMetrics(options.Thresholds)
	client := http.NewClient(httpMetrics)

	pool, err := virtualuser.CreatePool(maxVuCount, scriptContent, client)
	if err != nil {
		return nil, fmt.Errorf("error creating user pool: %w", err)
	}

	return &Engine{
		pool:     pool,
		options:  options,
		metrics:  httpMetrics,
		taskChan: make(chan struct{}, maxVuCount),
	}, nil
}

// runStage runs a stage with a given target number of virtual users
func (e *Engine) runStage(
	stage models.Stage,
	stageNumber int,
) error {
	log.Printf("Running stage %d with target %d for %s\n", stageNumber, stage.Target, stage.Duration)

	duration, rampUp, rampDown, err := stage.GetDurations()
	if err != nil {
		return fmt.Errorf("error getting durations: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	startUsers := int(atomic.LoadInt64(&e.activeUsers))
	endUsers := stage.Target

	var wg sync.WaitGroup

	// Ramp up and down users in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.rampUsers(ctx, startUsers, endUsers, rampUp, rampDown, duration)
	}()

	// Run virtual users
	for i := 0; i < endUsers; i++ {
		wg.Add(1)
		go e.runVirtualUser(ctx, &wg)
	}

	go e.displayStageProgress(ctx, stage, stageNumber)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Main loop to send tasks and check for context completion
	for {
		select {
		case <-ctx.Done():
			close(e.taskChan)
			wg.Wait()
			return nil
		case <-ticker.C:
			activeUsers := int(atomic.LoadInt64(&e.activeUsers))
			e.sendTasks(activeUsers)
		}
	}
}

// runVirtualUser runs a virtual user
func (e *Engine) runVirtualUser(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	user := e.pool.Fetch()
	defer e.pool.Return(user)

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-e.taskChan:
			if !ok {
				return
			}
			if err := user.Run(ctx); err != nil {
				log.Printf("Error running virtual user: %v", err)
			}
		}
	}
}

// sendTasks sends tasks to virtual users
func (e *Engine) sendTasks(activeUsers int) {
	for i := 0; i < activeUsers; i++ {
		select {
		case e.taskChan <- struct{}{}:
		default:
		}
	}
}

// rampUsers adjusts the number of virtual users over time
func (e *Engine) rampUsers(
	ctx context.Context,
	start, end int,
	rampUp, rampDown, totalDuration time.Duration,
) {
	steadyStateDuration := totalDuration - rampUp - rampDown

	e.adjustUserCount(ctx, start, end, rampUp)

	select {
	case <-ctx.Done():
		return
	case <-time.After(steadyStateDuration):
	}

	e.adjustUserCount(ctx, end, start, rampDown)
}

// adjustUserCount adjusts the number of virtual users over time
func (e *Engine) adjustUserCount(
	ctx context.Context,
	start, end int,
	duration time.Duration,
) {
	if duration == 0 {
		atomic.StoreInt64(&e.activeUsers, int64(end))
		return
	}

	steps := int(duration.Seconds() * 10)
	stepSize := float64(end-start) / float64(steps)
	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()

	for i := 0; i < steps; i++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentUsers := int(math.Round(float64(start) + stepSize*float64(i)))
			atomic.StoreInt64(&e.activeUsers, int64(currentUsers))
		}
	}

	atomic.StoreInt64(&e.activeUsers, int64(end))
}

// displayStageProgress displays the stage progress with animations
func (e *Engine) displayStageProgress(
	ctx context.Context,
	stage models.Stage,
	stageNumber int,
) {
	duration, _ := time.ParseDuration(stage.Duration)
	startTime := time.Now()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed >= duration {
				fmt.Printf(
					"\rRunning stage %d [%s] %s / %s\n",
					stageNumber,
					strings.Repeat("=", progressBarLength),
					stage.Duration,
					stage.Duration,
				)
				return
			}

			progress := float64(elapsed) / float64(duration)
			bar := int(progress * progressBarLength)
			fmt.Printf(
				"\rRunning stage %d [%s%s] %ds / %s active vu: %d",
				stageNumber,
				strings.Repeat("=", bar),
				strings.Repeat("-", progressBarLength-bar),
				int(elapsed.Seconds()),
				stage.Duration,
				atomic.LoadInt64(&e.activeUsers),
			)
		}
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
