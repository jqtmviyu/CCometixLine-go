package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCountEnvironmentCountsMemoryMCPPluginsAndSkills(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", home)

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
	writeFile(t, filepath.Join(home, "skills", "user-skill", "skill.md"), "cmd")
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
	writeFile(t, filepath.Join(project, ".claude", "skills", "user-skill", "duplicate.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "project-skill", "skill.md"), "cmd")

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
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeFile(t, filepath.Join(home, "CLAUDE.md"), "global")
	writeFile(t, filepath.Join(home, "rules", "a.md"), "rule")

	counts := CountEnvironment(home)
	if counts.MemoryFiles != 2 {
		t.Fatalf("expected 2 memory files, got %d", counts.MemoryFiles)
	}
}

func TestCountEnvironmentSkillsDoNotDependOnPluginState(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", home)

	writeFile(t, filepath.Join(home, "skills", "user-skill", "skill.md"), "cmd")
	writeFile(t, filepath.Join(home, "skills", "root.md"), "cmd")
	writeFile(t, filepath.Join(project, ".claude", "skills", "project-skill", "one.md"), "cmd")
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

func TestConfiguredEffortLevelUsesUserSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "low"})

	level := ConfiguredEffortLevel(filepath.Join(t.TempDir(), "project"))
	if level != "low" {
		t.Fatalf("expected user effort low, got %q", level)
	}
}

func TestConfiguredEffortLevelProjectOverridesUser(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	t.Setenv("CLAUDE_CONFIG_DIR", home)
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
	t.Setenv("CLAUDE_CONFIG_DIR", home)
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
	t.Setenv("CLAUDE_CONFIG_DIR", home)
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
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "   "})
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{"effortLevel": "high"})

	level := ConfiguredEffortLevel(project)
	if level != "high" {
		t.Fatalf("expected fallback project effort high, got %q", level)
	}
}

func TestConfiguredEffortLevelAvoidsOverlappingClaudeDirDoubleRead(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	writeJSON(t, filepath.Join(home, "settings.json"), map[string]any{"effortLevel": "medium"})
	writeJSON(t, filepath.Join(home, ".claude", "settings.local.json"), map[string]any{"effortLevel": "high"})

	level := ConfiguredEffortLevel(home)
	if level != "high" {
		t.Fatalf("expected overlapping local effort high, got %q", level)
	}
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
