package main

type Options struct {
	Thresholds map[string][]string `json:"thresholds"`
	Stages     []Stage             `json:"stages"`
}

type Stage struct {
	Duration string `json:"duration"`
	Target   int    `json:"target"`
}

func main() {

}
