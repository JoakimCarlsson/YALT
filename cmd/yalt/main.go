package main

import (
	"github.com/joakimcarlsson/yalt/internal/engine"
)

func main() {
	scriptFile := "C:\\Users\\JCarlsson\\Documents\\Test\\test.js" //todo read from command line

	runtime := engine.New(scriptFile)

	err := runtime.Run()
	if err != nil {
		panic(err)
	}
}
