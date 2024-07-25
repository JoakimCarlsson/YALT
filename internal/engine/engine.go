package engine

import (
	"context"
	"fmt"
	"log"
	"math"
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
	pool         *virtualuser.UserPool
	options      *models.Options
	metrics      *metrics.Metrics
	activeUsers  int
	mu           sync.Mutex
	userChannels []chan struct{}
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
	log.Printf("Running stage %d with target %d for %s\n", stageNumber, stage.Target, stage.Duration)

	duration, err := time.ParseDuration(stage.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	rampUp, err := time.ParseDuration(stage.RampUp)
	if err != nil {
		rampUp = 0
	}

	rampDown, err := time.ParseDuration(stage.RampDown)
	if err != nil {
		rampDown = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	startUsers := e.getCurrentActiveUsers()
	endUsers := stage.Target

	e.userChannels = make([]chan struct{}, endUsers)
	for i := range e.userChannels {
		e.userChannels[i] = make(chan struct{}, 1)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		e.rampUsers(ctx, startUsers, endUsers, rampUp, rampDown, duration)
	}()

	for i := 0; i < endUsers; i++ {
		wg.Add(1)
		go e.runVirtualUser(ctx, &wg, i)
	}

	go e.displayStageProgress(stage, stageNumber)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			for _, ch := range e.userChannels {
				close(ch)
			}
			wg.Wait()
			return nil
		case <-ticker.C:
			activeUsers := e.getCurrentActiveUsers()
			e.sendTasks(activeUsers)
		}
	}
}

// runVirtualUser runs a virtual user
func (e *Engine) runVirtualUser(
	ctx context.Context,
	wg *sync.WaitGroup,
	index int,
) {
	defer wg.Done()
	user := e.pool.Fetch()
	defer e.pool.Return(user)

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.userChannels[index]:
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
		case e.userChannels[i] <- struct{}{}:
		default:
		}
	}
}

// ramUsers adjusts the number of virtual users over time
func (e *Engine) rampUsers(
	ctx context.Context,
	start, end int,
	rampUp, rampDown, totalDuration time.Duration,
) {
	steadyStateDuration := totalDuration - rampUp - rampDown

	e.adjustUserCount(ctx, start, end, rampUp)

	steadyStateCtx, cancel := context.WithTimeout(ctx, steadyStateDuration)
	<-steadyStateCtx.Done()
	cancel()

	e.adjustUserCount(ctx, end, start, rampDown)
}

// adjustUserCount adjusts the number of virtual users over time
func (e *Engine) adjustUserCount(
	ctx context.Context,
	start, end int,
	duration time.Duration,
) {
	if duration == 0 {
		e.setUserCount(end)
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
			e.setUserCount(currentUsers)
		}
	}

	e.setUserCount(end)
}

// setUserCount sets the number of active users
func (e *Engine) setUserCount(count int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activeUsers = count
	log.Printf("Active users: %d", e.activeUsers)
}

// getCurrentActiveUsers returns the current number of active users
func (e *Engine) getCurrentActiveUsers() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.activeUsers
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
			e.getCurrentActiveUsers(),
		)

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
