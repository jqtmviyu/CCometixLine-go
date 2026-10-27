package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCountEnvironmentCountsMemoryMCPPluginsAndSkills(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}
	setClaudeConfigDir(t, home)

	writeFile(t, filepath.Join(home, "CLAUDE.md"), "global")
	writeFile(t, filepath.Join(home, "rules", "a.md"), "rule")
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{
		"enabledMcpjsonServers": []any{"user-a", "shared"},
		"enabledPlugins": map[string]bool{
			"plugin-a@test": true,
			"plugin-b@test": false,
		},
	})
	writeJSON(t, home+".json", map[string]any{})
	writeFile(t, filepath.Join(home, "skills", "ignored.md"), "cmd")
	writeFile(t, filepath.Join(home, "skills", "user-skill", "SKILL.md"), "cmd")
	writeFile(t, filepath.Join(home, "skills", "user-skill", "extra.md"), "cmd")

	writeFile(t, filepath.Join(project, "CLAUDE.md"), "project")
	writeFile(t, filepath.Join(project, "CLAUDE.local.md"), "project-local")
	writeFile(t, filepath.Join(project, ".claude", "CLAUDE.md"), "project-claude")
	writeFile(t, filepath.Join(project, ".claude", "CLAUDE.local.md"), "project-claude-local")
	writeFile(t, filepath.Join(project, ".claude", "rules", "b.md"), "rule")
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{
		"enabledMcpjsonServers": []any{"project-a", "shared"},
		"enabledPlugins":        map[string]bool{"plugin-c@test": true, "plugin-a@test": false},
	})
	writeJSON(t, filepath.Join(project, ".claude", "settings.local.json"), map[string]any{
		"enabledMcpjsonServers": []any{"project-b", "shared"},
		"enabledPlugins":        map[string]bool{"plugin-d@test": true, "plugin-c@test": true},
	})
	writeFile(t, filepath.Join(project, ".claude", "skills", "ignored.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "user-skill", "SKILL.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "project-skill", "SKILL.md"), "cmd")

	counts := CountEnvironment(project)
	if counts.MemoryFiles != 7 {
		t.Fatalf("expected 7 memory files, got %d", counts.MemoryFiles)
	}
	if counts.MCPs != 4 {
		t.Fatalf("expected deduplicated mcps=4, got %d", counts.MCPs)
	}
	if counts.Skills != 2 {
		t.Fatalf("expected deduplicated skills=2, got %d", counts.Skills)
	}
	if counts.Plugins != 3 {
		t.Fatalf("expected deduplicated plugins=3, got %d", counts.Plugins)
	}
}

func TestCountEnvironmentAvoidsOverlappingClaudeDirDoubleCount(t *testing.T) {
	home := t.TempDir()
	setClaudeConfigDir(t, home)
	writeFile(t, filepath.Join(home, "CLAUDE.md"), "global")
	writeFile(t, filepath.Join(home, "rules", "a.md"), "rule")

	counts := CountEnvironment(home)
	if counts.MemoryFiles != 2 {
		t.Fatalf("expected 2 memory files, got %d", counts.MemoryFiles)
	}
}

func TestEnvironmentCacheRefreshesAfterTTLAndKeepsEffortCurrent(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	setClaudeConfigDir(t, home)
	writeFile(t, filepath.Join(home, "CLAUDE.md"), "global")
	writeFile(t, filepath.Join(home, "skills", "first", "SKILL.md"), "skill")
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{
		"effortLevel":           "low",
		"enabledMcpjsonServers": []any{"mcp-a", "mcp-b"},
		"enabledPlugins":        map[string]bool{"plugin-a": true, "plugin-b": true},
	})

	now := time.Now().Truncate(time.Second)
	first := loadSettingsSnapshotAt(project, settingsSnapshotOptions{includeEnvironment: true, includeEffort: true, useCache: true}, now)
	if first.EnvironmentCounts() != (EnvironmentCounts{MemoryFiles: 1, MCPs: 2, Skills: 1, Plugins: 2}) || first.ConfiguredEffortLevel() != "low" {
		t.Fatalf("unexpected initial snapshot: %#v", first)
	}

	writeFile(t, filepath.Join(project, "CLAUDE.md"), "project")
	writeFile(t, filepath.Join(home, "skills", "second", "SKILL.md"), "skill")
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{
		"effortLevel":           "high",
		"enabledMcpjsonServers": []any{"mcp-a"},
		"enabledPlugins":        map[string]bool{"plugin-a": true},
	})

	cached := loadSettingsSnapshotAt(project, settingsSnapshotOptions{includeEnvironment: true, includeEffort: true, useCache: true}, now.Add(environmentCacheTTL-time.Second))
	if cached.EnvironmentCounts() != first.EnvironmentCounts() || cached.ConfiguredEffortLevel() != "high" {
		t.Fatalf("cache should retain counts while effort stays fresh: %#v", cached)
	}

	refreshed := loadSettingsSnapshotAt(project, settingsSnapshotOptions{includeEnvironment: true, includeEffort: true, useCache: true}, now.Add(environmentCacheTTL+time.Second))
	if refreshed.EnvironmentCounts() != (EnvironmentCounts{MemoryFiles: 2, MCPs: 1, Skills: 2, Plugins: 1}) || refreshed.ConfiguredEffortLevel() != "high" {
		t.Fatalf("unexpected refreshed snapshot: %#v", refreshed)
	}
}

func TestEnvironmentCacheSeparatesConfigAndWorkingDirectory(t *testing.T) {
	sharedBridgeDir := t.TempDir()
	homeA := t.TempDir()
	homeB := t.TempDir()
	projectA := filepath.Join(t.TempDir(), "project-a")
	projectB := filepath.Join(t.TempDir(), "project-b")
	t.Setenv("CCLINE_BRIDGE_DIR", sharedBridgeDir)
	writeFile(t, filepath.Join(homeA, "CLAUDE.md"), "home-a")
	writeFile(t, filepath.Join(projectB, "CLAUDE.md"), "project-b")
	now := time.Now().Truncate(time.Second)

	t.Setenv("CLAUDE_CONFIG_DIR", homeA)
	countsA := loadSettingsSnapshotAt(projectA, settingsSnapshotOptions{includeEnvironment: true, useCache: true}, now).EnvironmentCounts()
	t.Setenv("CLAUDE_CONFIG_DIR", homeB)
	countsB := loadSettingsSnapshotAt(projectA, settingsSnapshotOptions{includeEnvironment: true, useCache: true}, now).EnvironmentCounts()
	projectCounts := loadSettingsSnapshotAt(projectB, settingsSnapshotOptions{includeEnvironment: true, useCache: true}, now).EnvironmentCounts()

	if countsA.MemoryFiles != 1 || countsB.MemoryFiles != 0 || projectCounts.MemoryFiles != 1 {
		t.Fatalf("cache scope mixed config or working directories: A=%#v B=%#v project=%#v", countsA, countsB, projectCounts)
	}
}

func TestUniquePathsDeduplicatesSymlinkAliases(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if paths := uniquePaths([]string{root, alias}); len(paths) != 1 {
		t.Fatalf("expected one unique path, got %#v", paths)
	}
}

func TestCountEnvironmentSkillsDoNotDependOnPluginState(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}
	setClaudeConfigDir(t, home)

	writeFile(t, filepath.Join(home, "skills", "user-skill", "SKILL.md"), "cmd")
	writeFile(t, filepath.Join(home, "skills", "root.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "project-skill", "SKILL.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "project-skill", "two.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "ignored.md"), "cmd")

	counts := CountEnvironment(project)
	if counts.Skills != 2 {
		t.Fatalf("expected deduplicated skills=2, got %d", counts.Skills)
	}
	if counts.MCPs != 0 {
		t.Fatalf("expected mcps=0 when enabledMcpjsonServers missing, got %d", counts.MCPs)
	}
	if counts.Plugins != 0 {
		t.Fatalf("expected plugins=0 when enabledPlugins missing, got %d", counts.Plugins)
	}
}

func TestCountEnvironmentIncludesSymlinkedUserSkill(t *testing.T) {
	home := t.TempDir()
	setClaudeConfigDir(t, home)
	target := t.TempDir()
	writeFile(t, filepath.Join(target, "SKILL.md"), "skill")
	if err := os.MkdirAll(filepath.Join(home, "skills"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(home, "skills", "linked-skill")); err != nil {
		t.Fatal(err)
	}
	if counts := CountEnvironment(t.TempDir()); counts.Skills != 1 {
		t.Fatalf("expected 1 symlinked user skill, got %d", counts.Skills)
	}
}

func TestCountEnvironmentFiltersModelDisabledSkills(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	setClaudeConfigDir(t, home)
	for _, root := range []string{home, filepath.Join(project, ".claude")} {
		writeFile(t, filepath.Join(root, "skills", "disabled", "SKILL.md"), "---\ndisable-model-invocation: true # manual only\n---\nBody")
		writeFile(t, filepath.Join(root, "skills", "enabled", "SKILL.md"), "---\ndisable-model-invocation: false\n---\nBody")
		writeFile(t, filepath.Join(root, "skills", "default", "SKILL.md"), "---\nname: default\n---\ndisable-model-invocation: true")
		writeFile(t, filepath.Join(root, "skills", "body-only", "SKILL.md"), "Example:\ndisable-model-invocation: true")
		writeFile(t, filepath.Join(root, "skills", "no-entry", "notes.md"), "notes")
	}
	target := t.TempDir()
	writeFile(t, filepath.Join(target, "SKILL.md"), "---\ndisable-model-invocation: true\n---\nBody")
	if err := os.Symlink(target, filepath.Join(home, "skills", "linked-disabled")); err != nil {
		t.Fatal(err)
	}
	if counts := CountEnvironment(project); counts.Skills != 3 {
		t.Fatalf("expected 3 callable skills after filtering and deduplication, got %d", counts.Skills)
	}
}

func TestConfiguredEffortLevelUsesUserSettings(t *testing.T) {
	home := t.TempDir()
	setClaudeConfigDir(t, home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "low"})

	level := ConfiguredEffortLevel(filepath.Join(t.TempDir(), "project"))
	if level != "low" {
		t.Fatalf("expected user effort low, got %q", level)
	}
}

func TestConfiguredEffortLevelProjectOverridesUser(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	setClaudeConfigDir(t, home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "low"})
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	level := ConfiguredEffortLevel(project)
	if level != "high" {
		t.Fatalf("expected project effort high, got %q", level)
	}
}

func TestConfiguredEffortLevelLocalOverridesProject(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	setClaudeConfigDir(t, home)
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "medium"})
	writeJSON(t, filepath.Join(project, ".claude", "settings.local.json"), map[string]any{"effortLevel": "xhigh"})

	level := ConfiguredEffortLevel(project)
	if level != "xhigh" {
		t.Fatalf("expected local effort xhigh, got %q", level)
	}
}

func TestConfiguredEffortLevelIgnoresInvalidJSON(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	setClaudeConfigDir(t, home)
	writeFile(t, filepath.Join(home, "settings.json"), "{")
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	level := ConfiguredEffortLevel(project)
	if level != "high" {
		t.Fatalf("expected project effort high after invalid user json, got %q", level)
	}
}

func TestConfiguredEffortLevelIgnoresWhitespace(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	setClaudeConfigDir(t, home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "   "})
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	level := ConfiguredEffortLevel(project)
	if level != "high" {
		t.Fatalf("expected fallback project effort high, got %q", level)
	}
}

func TestConfiguredEffortLevelAvoidsOverlappingClaudeDirDoubleRead(t *testing.T) {
	home := t.TempDir()
	setClaudeConfigDir(t, home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "medium"})
	writeJSON(t, filepath.Join(home, ".claude", "settings.local.json"), map[string]any{"effortLevel": "high"})

	level := ConfiguredEffortLevel(home)
	if level != "high" {
		t.Fatalf("expected overlapping local effort high, got %q", level)
	}
}

func setClaudeConfigDir(t *testing.T, path string) {
	t.Helper()
	t.Setenv("CLAUDE_CONFIG_DIR", path)
	t.Setenv("CCLINE_BRIDGE_DIR", filepath.Join(path, "ccline", "bridge"))
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}
