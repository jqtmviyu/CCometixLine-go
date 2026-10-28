package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestRunUnknownCommandShowsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unknown"}, strings.NewReader("ignored"), &stdout, &stderr, false); code != 2 {
		t.Fatalf("expected usage error, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: ccline [preview]") {
		t.Fatalf("unexpected usage: %q", stderr.String())
	}
}
