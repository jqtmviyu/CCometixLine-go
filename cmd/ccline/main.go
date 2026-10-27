package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"ccometixline-go/internal/companion"
	"ccometixline-go/internal/config"
	"ccometixline-go/internal/preview"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/render"
	"ccometixline-go/internal/segments"
	"ccometixline-go/internal/state"
)

const maxInputBytes = 1 << 20

type taskLine struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

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
	return runWithMod(args, stdin, stdout, stderr, terminal, companion.Run)
}

func runWithMod(args []string, stdin io.Reader, stdout, stderr io.Writer, terminal bool, runMod func([]string, io.Writer, io.Writer) int) int {
	if len(args) > 0 && args[0] == "mod" {
		return runMod(args[1:], stdout, stderr)
	}

	previewRequested := len(args) == 1 && args[0] == "preview"
	if len(args) > 0 && !previewRequested {
		fmt.Fprintln(stderr, "usage: ccline [preview|mod <install|status|uninstall>]")
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

	taskInput, isTaskInput, err := protocol.DecodeSubagentInput(data)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	generator := render.StatusLineGenerator{Config: cfg}
	if isTaskInput {
		if err := writeTaskLines(stdout, generator, cfg, *taskInput, time.Now()); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	var input protocol.InputData
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	modelConfig := config.LoadModelConfig()
	bridge := state.ReadBridge(input.SessionID, time.Now())
	collected := segments.CollectAllWithBridge(cfg, modelConfig, input, bridge)
	fmt.Fprintln(stdout, generator.Generate(collected))
	return 0
}

func writeTaskLines(writer io.Writer, generator render.StatusLineGenerator, cfg config.Config, input protocol.SubagentInput, now time.Time) error {
	encoder := json.NewEncoder(writer)
	for _, task := range input.Tasks {
		items := segments.CollectTask(cfg, task, now)
		content := generator.GenerateTaskLine(items, input.Columns)
		if content == "" {
			continue
		}
		if err := encoder.Encode(taskLine{ID: task.ID, Content: content}); err != nil {
			return err
		}
	}
	return nil
}
