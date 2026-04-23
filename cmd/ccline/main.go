package main

import (
	"encoding/json"
	"fmt"
	"os"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/render"
	"ccometixline-go/internal/segments"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	modelConfig := config.LoadModelConfig()

	var input protocol.InputData
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	collected := segments.CollectAll(cfg, modelConfig, input)
	generator := render.StatusLineGenerator{Config: cfg}
	fmt.Println(generator.Generate(collected))
}
