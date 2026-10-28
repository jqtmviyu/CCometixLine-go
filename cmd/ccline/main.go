package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/preview"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/render"
	"ccometixline-go/internal/segments"
)

const maxInputBytes = 1 << 20

func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, stdinIsTerminal(os.Stdin))
	if code != 0 {
		os.Exit(code)
	}
}

func stdinIsTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, terminal bool) int {
	previewRequested := len(args) == 1 && args[0] == "preview"
	if len(args) > 0 && !previewRequested {
		fmt.Fprintln(stderr, "usage: ccline [preview]")
		return 2
	}

	if previewRequested || len(args) == 0 && terminal {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(stderr, "load ccline config: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, preview.Render(cfg))
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	data, err := io.ReadAll(io.LimitReader(stdin, maxInputBytes+1))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(data) > maxInputBytes {
		fmt.Fprintln(stderr, "statusline input exceeds size limit")
		return 1
	}

	layout := cfg.ResolveLayout()
	generator := render.StatusLineGenerator{Style: cfg.Style, Layout: layout}

	var input protocol.InputData
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	modelConfig := config.LoadModelConfig()
	collector := segments.Collector{Layout: layout, Cache: cfg.Cache, Models: modelConfig}
	collected := collector.Collect(input)
	fmt.Fprintln(stdout, generator.Generate(collected))
	return 0
}
