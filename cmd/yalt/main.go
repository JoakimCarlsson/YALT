package main

import (
	"flag"
	"github.com/joakimcarlsson/yalt/internal/engine"
	"log"
)

func main() {
	scriptFile := "C:\\Users\\JCarlsson\\Documents\\Test\\test.js"
	flag.Parse()

	runtime, err := engine.New(scriptFile)
	if err != nil {
		log.Fatalf("Error creating engine: %v", err)
	}

	if err := runtime.Run(); err != nil {
		log.Fatalf("Error running the engine: %v", err)
	}
}
