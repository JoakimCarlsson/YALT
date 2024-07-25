package models

import (
	"fmt"
	"time"
)

type Stage struct {
	Target   int    `json:"target"`
	Duration string `json:"duration"`
	RampUp   string `json:"rampUp,omitempty"`
	RampDown string `json:"rampDown,omitempty"`
}

// GetDurations returns the duration, rampUp and rampDown as time.Duration
func (s *Stage) GetDurations() (
	duration, rampUp, rampDown time.Duration,
	err error,
) {
	duration, err = time.ParseDuration(s.Duration)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid duration format: %w", err)
	}

	rampUp, err = time.ParseDuration(s.RampUp)
	if err != nil {
		rampUp = 0
	}

	rampDown, err = time.ParseDuration(s.RampDown)
	if err != nil {
		rampDown = 0
	}

	return duration, rampUp, rampDown, nil
}
