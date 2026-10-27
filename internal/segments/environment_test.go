package segments

import (
	"os"
	"path/filepath"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestEnvironmentSegmentCollectReturnsNilForEmptyEnvironment(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	workspace := t.TempDir()
	segment := EnvironmentSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: workspace}}, Settings: config.LoadSettingsSnapshot(workspace)}

	data := segment.Collect()
	if data != nil {
		t.Fatalf("expected nil data for empty environment, got %#v", data)
	}
}

func TestEnvironmentSegmentFormatsCountsWithMiddleDots(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	workspace := t.TempDir()
	writeEnvTestFile(t, filepath.Join(workspace, ".claude", "skills", "demo", "SKILL.md"), "cmd")
	segment := EnvironmentSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: workspace}}, Settings: config.LoadSettingsSnapshot(workspace)}

	data := segment.Collect()
	if data == nil || data.Primary != "memory 0 · skill 1 · mcp 0 · plugin 0" {
		t.Fatalf("unexpected primary: %#v", data)
	}
}

func writeEnvTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
