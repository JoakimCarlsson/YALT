package models

type Options struct {
	Thresholds map[string][]string `json:"thresholds"`
	Stages     []Stage             `json:"stages"`
}
