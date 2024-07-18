package main

import (
	"flag"
	"fmt"
	"github.com/joakimcarlsson/yalt/internal/engine"
	"log"
	"os"
)

func main() {
	scriptFile := flag.String("script", "", "Path to the script file")
	flag.Parse()

	if *scriptFile == "" {
		fmt.Println("Usage: go run main.go -script=path/to/your/script.js")
		os.Exit(1)
	}

	runtime := engine.New(*scriptFile)

	if err := runtime.Run(); err != nil {
		log.Fatalf("Error running the engine: %v", err)
	}
}
