package segments

import (
	"os"
	"path/filepath"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

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
	items := (Collector{Layout: cfg.ResolveLayout(), Cache: cfg.Cache}).Collect(input)
	if len(items) != 1 || items[0].Config.ID != config.SegmentModel {
		t.Fatalf("expected only the selected model segment, got %#v", items)
	}
	cacheDir := filepath.Join(cacheRoot, "environment")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("unselected environment segment created a cache: %v", err)
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
		for _, item := range (Collector{Layout: cfg.ResolveLayout(), Cache: cfg.Cache}).Collect(input) {
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
