package segments

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ccometixline-go/internal/protocol"
)

func TestEffortSegmentCollectFromPayload(t *testing.T) {
	isolateClaudeConfig(t)
	t.Setenv(effortEnvKey, "")
	segment := EffortSegment{Input: protocol.InputData{Effort: "HIGH"}}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "high" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestEffortSegmentCollectUnknownPayload(t *testing.T) {
	isolateClaudeConfig(t)
	t.Setenv(effortEnvKey, "")
	segment := EffortSegment{Input: protocol.InputData{Effort: "super-max"}}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "super-max?" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestEffortSegmentCollectDefault(t *testing.T) {
	isolateClaudeConfig(t)
	t.Setenv(effortEnvKey, "")
	segment := EffortSegment{Input: protocol.InputData{Effort: "@@"}}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "auto" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestResolveEffortPrefersEnv(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".claude", "settings.local.json"), map[string]any{"effortLevel": "max"})
	t.Setenv(effortEnvKey, "high")

	effort := resolveEffort(protocol.InputData{Effort: "medium", Workspace: protocol.Workspace{CurrentDir: dir}})
	if effort != "high" {
		t.Fatalf("expected env effort high, got %q", effort)
	}
}

func TestResolveEffortPrefersPayloadOverSettings(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "medium"})

	effort := resolveEffort(protocol.InputData{Effort: "high", Workspace: protocol.Workspace{CurrentDir: dir}})
	if effort != "high" {
		t.Fatalf("expected payload effort high, got %q", effort)
	}
}

func TestResolveEffortIgnoresInvalidEnv(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".claude", "settings.local.json"), map[string]any{"effortLevel": "max"})
	t.Setenv(effortEnvKey, "@@")

	effort := resolveEffort(protocol.InputData{Workspace: protocol.Workspace{CurrentDir: dir}})
	if effort != "max" {
		t.Fatalf("expected settings effort max, got %q", effort)
	}
}

func TestResolveEffortEnvAutoResetsToDefault(t *testing.T) {
	t.Setenv(effortEnvKey, "auto")

	effort := resolveEffort(protocol.InputData{})
	if effort != "" {
		t.Fatalf("expected empty effort for auto, got %q", effort)
	}
}

func TestResolveEffortFromUserSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "high"})

	effort := resolveEffort(protocol.InputData{})
	if effort != "high" {
		t.Fatalf("expected user settings effort high, got %q", effort)
	}
}

func TestResolveEffortProjectSettingsOverrideUserSettings(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "medium"})
	writeJSON(t, filepath.Join(dir, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})
	writeJSON(t, filepath.Join(dir, ".claude", "settings.local.json"), map[string]any{"effortLevel": "max"})

	effort := resolveEffort(protocol.InputData{Workspace: protocol.Workspace{CurrentDir: dir}})
	if effort != "max" {
		t.Fatalf("expected project local settings effort max, got %q", effort)
	}
}

func TestResolveEffortIgnoresInvalidSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "@@"})

	effort := resolveEffort(protocol.InputData{})
	if effort != "" {
		t.Fatalf("expected default effort, got %q", effort)
	}
}

func isolateClaudeConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	return home
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
