package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/render"
)

type forbiddenReader struct {
	called bool
}

func (r *forbiddenReader) Read([]byte) (int, error) {
	r.called = true
	return 0, errors.New("stdin must not be read")
}

func TestRunBareTerminalShowsPreviewWithoutReadingStdin(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	stdin := &forbiddenReader{}
	var stdout, stderr bytes.Buffer
	if code := run(nil, stdin, &stdout, &stderr, true); code != 0 {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}
	if stdin.called {
		t.Fatal("preview read stdin")
	}
	if !strings.Contains(stdout.String(), "配置预览") || !strings.Contains(stdout.String(), "statusLine") {
		t.Fatalf("unexpected preview output: %q", stdout.String())
	}
}

func TestRunPreviewCommandIgnoresPipedStdin(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	stdin := &forbiddenReader{}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"preview"}, stdin, &stdout, &stderr, false); code != 0 {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}
	if stdin.called || !strings.Contains(stdout.String(), "配置预览") {
		t.Fatalf("preview should ignore stdin and render config: %q", stdout.String())
	}
}

func TestRunPipeKeepsStatuslineProtocol(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	cwd := t.TempDir()
	input := `{"model":{"id":"claude-sonnet-4-5-20250929","display_name":"Sonnet 4.5"},"workspace":{"current_dir":"` + cwd + `"}}`
	var stdout, stderr bytes.Buffer
	if code := run(nil, strings.NewReader(input), &stdout, &stderr, false); code != 0 {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Sonnet 4.5") || strings.Contains(stdout.String(), "配置预览") {
		t.Fatalf("pipe did not use the statusline protocol: %q", stdout.String())
	}
}

func TestRunInvalidPipedInputDoesNotFallBackToPreview(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := run(nil, strings.NewReader("not-json"), &stdout, &stderr, false); code == 0 {
		t.Fatal("expected malformed statusline input to fail")
	}
	if strings.Contains(stdout.String(), "配置预览") {
		t.Fatalf("malformed input unexpectedly entered preview: %q", stdout.String())
	}
}

func TestRunPreviewReportsInvalidConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	configDir := filepath.Join(root, "ccline")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[layout\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"preview"}, strings.NewReader("ignored"), &stdout, &stderr, false); code == 0 {
		t.Fatal("expected invalid config to fail preview")
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "config") {
		t.Fatalf("unexpected invalid-config handling: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestStdinIsTerminalRejectsPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	if stdinIsTerminal(reader) {
		t.Fatal("pipe was detected as a terminal")
	}
}

func TestRunModCommandDoesNotReadStdin(t *testing.T) {
	stdin := &forbiddenReader{}
	var stdout, stderr bytes.Buffer
	called := false
	commandArgs := []string{"install"}
	code := runWithMod(
		append([]string{"mod"}, commandArgs...),
		stdin,
		&stdout,
		&stderr,
		false,
		func(args []string, out io.Writer, errOut io.Writer) int {
			called = true
			if !reflect.DeepEqual(args, commandArgs) {
				t.Errorf("unexpected mod args: %#v", args)
			}
			return 0
		},
	)
	if code != 0 || !called || stdin.called {
		t.Fatalf("mod command did not route independently: code=%d called=%v stdinRead=%v stderr=%q", code, called, stdin.called, stderr.String())
	}
}

func TestRunUnknownCommandShowsModUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unknown"}, strings.NewReader("ignored"), &stdout, &stderr, false); code != 2 {
		t.Fatalf("expected usage error, got %d", code)
	}
	if !strings.Contains(stderr.String(), "mod") {
		t.Fatalf("usage does not include mod command: %q", stderr.String())
	}
}

func TestWriteTaskLinesEmitsTaskJSON(t *testing.T) {
	cfg := config.DefaultConfig()
	input := protocol.SubagentInput{
		Columns: 80,
		Tasks: []protocol.Task{{
			ID:     "a1",
			Status: "running",
			Label:  "Read \"config\"",
		}},
	}
	var output bytes.Buffer
	if err := writeTaskLines(&output, render.StatusLineGenerator{Config: cfg}, cfg, input, time.Now()); err != nil {
		t.Fatal(err)
	}

	var line taskLine
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &line); err != nil {
		t.Fatal(err)
	}
	if line.ID != "a1" || !strings.Contains(line.Content, `Read "config"`) {
		t.Fatalf("unexpected task output: %#v", line)
	}
}
