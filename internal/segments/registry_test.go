package segments

import (
	"os"
	"path/filepath"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestLoadSettingsSnapshotSkipsDisabledSettingsSegments(t *testing.T) {
	configDir := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	cfg := config.Config{Segments: []config.SegmentConfig{{ID: config.SegmentModel, Enabled: true}}}
	snapshot := loadSettingsSnapshot(collectableSegments(cfg), cwd)
	if snapshot.EnvironmentCounts() != (config.EnvironmentCounts{}) || snapshot.ConfiguredEffortLevel() != "" {
		t.Fatalf("unexpected settings snapshot for disabled segments: %#v", snapshot)
	}
	cacheDir := filepath.Join(configDir, "ccline", "bridge", "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("disabled settings segments created an environment cache: %v", err)
	}
}

func TestLoadSettingsSnapshotForEffortSkipsEnvironmentScan(t *testing.T) {
	configDir := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{"effortLevel":"high"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Segments: []config.SegmentConfig{{ID: config.SegmentEffort, Enabled: true}}}
	snapshot := loadSettingsSnapshot(collectableSegments(cfg), cwd)
	if snapshot.ConfiguredEffortLevel() != "high" || snapshot.EnvironmentCounts() != (config.EnvironmentCounts{}) {
		t.Fatalf("unexpected effort-only snapshot: %#v", snapshot)
	}
	cacheDir := filepath.Join(configDir, "ccline", "bridge", "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("effort-only collection created an environment cache: %v", err)
	}
}

func TestCollectAllSkipsEnabledSegmentsOutsideLayout(t *testing.T) {
	configDir := t.TempDir()
	bridgeDir := filepath.Join(configDir, "bridge")
	cwd := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("CCLINE_BRIDGE_DIR", bridgeDir)

	cfg := config.DefaultConfig()
	cfg.Layout.Lines = []config.LayoutLine{{Segments: []config.SegmentID{config.SegmentModel}}}
	input := protocol.InputData{
		Model:     protocol.Model{DisplayName: "Sonnet"},
		Workspace: protocol.Workspace{CurrentDir: cwd},
	}
	items := CollectAll(cfg, config.ModelConfig{}, input)
	if len(items) != 1 || items[0].Config.ID != config.SegmentModel {
		t.Fatalf("expected only the selected model segment, got %#v", items)
	}
	cacheDir := filepath.Join(bridgeDir, "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("unselected environment segment created a cache: %v", err)
	}
}
