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
	snapshot := loadSettingsSnapshot(collectableSegments(cfg), protocol.InputData{Workspace: protocol.Workspace{CurrentDir: cwd}}, cfg.Cache)
	if snapshot.EnvironmentCounts() != (config.EnvironmentCounts{}) || snapshot.ConfiguredEffortLevel() != "" {
		t.Fatalf("unexpected settings snapshot for disabled segments: %#v", snapshot)
	}
	cacheDir := filepath.Join(configDir, "ccline", "cache", "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("disabled settings segments created an environment cache: %v", err)
	}
}

func TestLoadSettingsSnapshotForEffortSkipsEnvironmentScan(t *testing.T) {
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	configDir := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{"effortLevel":"high"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Segments: []config.SegmentConfig{{ID: config.SegmentEffort, Enabled: true}}}
	snapshot := loadSettingsSnapshot(collectableSegments(cfg), protocol.InputData{Workspace: protocol.Workspace{CurrentDir: cwd}}, cfg.Cache)
	if snapshot.ConfiguredEffortLevel() != "high" || snapshot.EnvironmentCounts() != (config.EnvironmentCounts{}) {
		t.Fatalf("unexpected effort-only snapshot: %#v", snapshot)
	}
	cacheDir := filepath.Join(configDir, "ccline", "cache", "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("effort-only collection created an environment cache: %v", err)
	}
}

func TestCollectAllSkipsEnabledSegmentsOutsideLayout(t *testing.T) {
	configDir := t.TempDir()
	cacheRoot := filepath.Join(configDir, "cache")
	cwd := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("CCLINE_CACHE_DIR", cacheRoot)

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
	cacheDir := filepath.Join(cacheRoot, "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("unselected environment segment created a cache: %v", err)
	}
}

func TestSettingsReadEffortOnlyWhenFallbackIsNeeded(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"effortLevel":"low"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	selected := []config.SegmentConfig{{ID: config.SegmentEffort, Enabled: true}}
	for _, tc := range []struct{ env, payload, loaded, display string }{
		{"high", "max", "", "high"},
		{"auto", "max", "", "auto"},
		{"", "medium", "", "medium"},
		{"@@", "auto", "", "auto"},
		{"", "@@", "low", "low"},
		{"", "", "low", "low"},
	} {
		t.Run(tc.env+"/"+tc.payload, func(t *testing.T) {
			t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", tc.env)
			input := protocol.InputData{Workspace: protocol.Workspace{CurrentDir: t.TempDir()}, Effort: &protocol.Effort{Level: tc.payload}}
			snapshot := loadSettingsSnapshot(selected, input, config.DefaultCacheConfig())
			if got := snapshot.ConfiguredEffortLevel(); got != tc.loaded {
				t.Fatalf("settings effort=%q, want %q", got, tc.loaded)
			}
			data := (EffortSegment{Input: input, Settings: snapshot}).Collect()
			if data.Primary != tc.display {
				t.Fatalf("display=%q, want %q", data.Primary, tc.display)
			}
		})
	}
}

func TestCollectionUsesConfiguredCacheDurations(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv("CCLINE_CACHE_DIR", t.TempDir())
	cwd := t.TempDir()
	runGitCmd(t, cwd, "init", "-b", "main")
	cfg := config.DefaultConfig()
	cfg.Cache = config.CacheConfig{GitSeconds: 3600, EnvironmentSeconds: 3600}
	cfg.Layout.Lines = []config.LayoutLine{{Segments: []config.SegmentID{config.SegmentGit, config.SegmentEnvironment}}}
	input := protocol.InputData{Workspace: protocol.Workspace{CurrentDir: cwd}}
	collect := func() map[config.SegmentID]SegmentData {
		result := make(map[config.SegmentID]SegmentData)
		for _, item := range CollectAll(cfg, config.ModelConfig{}, input) {
			result[item.Config.ID] = item.Data
		}
		return result
	}
	first := collect()
	if first[config.SegmentGit].Metadata["status"] != string(GitStatusClean) {
		t.Fatalf("initial Git status: %#v", first)
	}
	if err := os.WriteFile(filepath.Join(cwd, "CLAUDE.md"), []byte("memory"), 0o600); err != nil {
		t.Fatal(err)
	}
	cached := collect()
	if cached[config.SegmentGit].Metadata["status"] != string(GitStatusClean) || cached[config.SegmentEnvironment].Primary != "" {
		t.Fatalf("configured caches were not used: %#v", cached)
	}
	cfg.Cache = config.CacheConfig{}
	current := collect()
	if current[config.SegmentGit].Metadata["status"] != string(GitStatusDirty) || current[config.SegmentEnvironment].Metadata["memory_files"] != "1" {
		t.Fatalf("zero TTL should refresh both segments: %#v", current)
	}
}
