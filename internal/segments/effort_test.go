package segments

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestEffortSegmentCollectFromPayload(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	segment := EffortSegment{Input: protocol.InputData{Effort: &protocol.Effort{Level: "HIGH"}}, Settings: config.LoadSettingsSnapshot(t.TempDir())}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "high" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestEffortSegmentCollectUnknownPayload(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	segment := EffortSegment{Input: protocol.InputData{Effort: &protocol.Effort{Level: "super-max"}}, Settings: config.LoadSettingsSnapshot(t.TempDir())}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "super-max?" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestEffortSegmentCollectDefault(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	segment := EffortSegment{Input: protocol.InputData{Effort: &protocol.Effort{Level: "@@"}}, Settings: config.LoadSettingsSnapshot(t.TempDir())}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "auto" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestResolveEffortUsesPayloadWhenNoOverridesExist(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	input := protocol.InputData{Effort: &protocol.Effort{Level: "high"}}

	effort := resolveEffort(input, config.LoadSettingsSnapshot(t.TempDir()))
	if effort != "high" {
		t.Fatalf("expected payload effort high, got %q", effort)
	}
}

func TestResolveEffortEnvironmentOverridesPayloadAndSettings(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "high")
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "low"})
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "medium"})

	input := protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "xhigh"},
	}

	effort := resolveEffort(input, config.LoadSettingsSnapshot(input.Workspace.CurrentDir))
	if effort != "high" {
		t.Fatalf("expected env effort high, got %q", effort)
	}
}

func TestResolveEffortPayloadOverridesSettings(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "xhigh"})

	input := protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "medium"},
	}

	effort := resolveEffort(input, config.LoadSettingsSnapshot(input.Workspace.CurrentDir))
	if effort != "medium" {
		t.Fatalf("expected payload effort medium, got %q", effort)
	}
}

func TestResolveEffortProjectLocalOverridesProjectSettings(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "medium"})
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.local.json"), map[string]any{"effortLevel": "high"})

	input := protocol.InputData{Workspace: protocol.Workspace{CurrentDir: project}}
	effort := resolveEffort(input, config.LoadSettingsSnapshot(input.Workspace.CurrentDir))
	if effort != "high" {
		t.Fatalf("expected local settings effort high, got %q", effort)
	}
}

func TestResolveEffortProjectSettingsOverrideUserSettings(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "low"})
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	input := protocol.InputData{Workspace: protocol.Workspace{CurrentDir: project}}
	effort := resolveEffort(input, config.LoadSettingsSnapshot(input.Workspace.CurrentDir))
	if effort != "high" {
		t.Fatalf("expected project settings effort high, got %q", effort)
	}
}

func TestResolveEffortAutoEnvironmentStopsFallback(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "auto")
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	segment := EffortSegment{Input: protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "max"},
	}, Settings: config.LoadSettingsSnapshot(project)}
	data := segment.Collect()
	if data.Primary != "auto" {
		t.Fatalf("expected env auto, got %q", data.Primary)
	}
}

func TestResolveEffortPayloadOverridesAutoSettings(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "auto"})

	segment := EffortSegment{Input: protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "high"},
	}, Settings: config.LoadSettingsSnapshot(project)}
	data := segment.Collect()
	if data.Primary != "high" {
		t.Fatalf("expected payload high, got %q", data.Primary)
	}
}

func TestResolveEffortPayloadWhenEnvironmentInvalid(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "@@")
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	input := protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "low"},
	}

	effort := resolveEffort(input, config.LoadSettingsSnapshot(input.Workspace.CurrentDir))
	if effort != "low" {
		t.Fatalf("expected payload effort low, got %q", effort)
	}
}

func TestResolveEffortPayloadWithInvalidSettings(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "@@"})

	input := protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "medium"},
	}

	effort := resolveEffort(input, config.LoadSettingsSnapshot(input.Workspace.CurrentDir))
	if effort != "medium" {
		t.Fatalf("expected fallback payload effort medium, got %q", effort)
	}
}

func TestResolveEffortEnvironmentOverridesAutoPayload(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "high")
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "medium"})

	segment := EffortSegment{Input: protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    &protocol.Effort{Level: "auto"},
	}, Settings: config.LoadSettingsSnapshot(project)}
	data := segment.Collect()
	if data.Primary != "high" {
		t.Fatalf("expected env high, got %q", data.Primary)
	}
}

func TestResolveEffortEnvironmentAutoStopsFallbackToSettings(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "auto")
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSONFile(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	segment := EffortSegment{Input: protocol.InputData{
		Workspace: protocol.Workspace{CurrentDir: project},
		Effort:    nil,
	}, Settings: config.LoadSettingsSnapshot(project)}
	data := segment.Collect()
	if data.Primary != "auto" {
		t.Fatalf("expected env auto, got %q", data.Primary)
	}
}

func TestResolveEffortUnknownValidEnvironmentValue(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "super-max")

	effort := resolveEffort(protocol.InputData{}, config.LoadSettingsSnapshot(t.TempDir()))
	if effort != "super-max?" {
		t.Fatalf("expected unknown env effort with question mark, got %q", effort)
	}
}

func TestResolveEffortMissingAllSourcesFallsBackToDefault(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	effort := resolveEffort(protocol.InputData{}, config.LoadSettingsSnapshot(t.TempDir()))
	if effort != "" {
		t.Fatalf("expected default effort, got %q", effort)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
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
