package models

type Stage struct {
	Target   int    `json:"target"`
	Duration string `json:"duration"`
	RampUp   string `json:"rampUp,omitempty"`
	RampDown string `json:"rampDown,omitempty"`
}
