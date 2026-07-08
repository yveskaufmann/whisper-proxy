package main

import (
	"fmt"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

func main() {
	var modelpath string // Path to the model
	var samples []float32 // Samples to process

	// Load the model
	model, err := whisper.New(modelpath)
	if err != nil {
		panic(err)
	}
	defer model.Close()

	// Process samples
	context, err := model.NewContext()
	if err != nil {
		panic(err)
	}
	if err := context.Process(samples, nil, nil, nil); err != nil {
		fmt.Printf("Failed to process samples: %v\n", err)
		return
	}

	// Print out the results
	for {
		segment, err := context.NextSegment()
		if err != nil {
			break
		}
		fmt.Printf("[%6s->%6s] %s\n", segment.Start, segment.End, segment.Text)
	}

}