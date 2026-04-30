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
		"mcpServers": map[string]any{"user-a": map[string]any{}},
		"enabledPlugins": map[string]bool{
			"plugin-a@test": true,
			"plugin-b@test": false,
		},
	})
	writeJSON(t, home+".json", map[string]any{
		"mcpServers":         map[string]any{"user-b": map[string]any{}},
		"disabledMcpServers": []any{"user-b"},
	})
	writeJSON(t, filepath.Join(home, "plugins", "installed_plugins.json"), map[string]any{
		"version": 2,
		"plugins": map[string]any{
			"plugin-a@test": []map[string]any{{"installPath": filepath.Join(home, "plugins", "cache", "plugin-a")}},
		},
	})
	writeJSON(t, filepath.Join(home, "plugins", "cache", "plugin-a", ".claude-plugin", "plugin.json"), map[string]any{
		"commands": []string{"./commands/one.md", "./commands/two.md"},
	})
	writeFile(t, filepath.Join(home, "skills", "user.md"), "cmd")

	writeFile(t, filepath.Join(project, "CLAUDE.md"), "project")
	writeFile(t, filepath.Join(project, "CLAUDE.local.md"), "project-local")
	writeFile(t, filepath.Join(project, ".claude", "CLAUDE.md"), "project-claude")
	writeFile(t, filepath.Join(project, ".claude", "CLAUDE.local.md"), "project-claude-local")
	writeFile(t, filepath.Join(project, ".claude", "rules", "b.md"), "rule")
	writeJSON(t, filepath.Join(project, ".claude", "settings.json"), map[string]any{
		"mcpServers": map[string]any{"project-a": map[string]any{}},
	})
	writeJSON(t, filepath.Join(project, ".claude", "settings.local.json"), map[string]any{
		"mcpServers":             map[string]any{"project-b": map[string]any{}},
		"disabledMcpjsonServers": []any{"project-mcp"},
	})
	writeJSON(t, filepath.Join(project, ".mcp.json"), map[string]any{
		"mcpServers": map[string]any{"project-mcp": map[string]any{}, "project-extra": map[string]any{}},
	})
	writeFile(t, filepath.Join(project, ".claude", "skills", "project.md"), "cmd")

	counts := CountEnvironment(project)
	if counts.MemoryFiles != 7 {
		t.Fatalf("expected 7 memory files, got %d", counts.MemoryFiles)
	}
	if counts.MCPs != 4 {
		t.Fatalf("expected 4 mcps, got %d", counts.MCPs)
	}
	if !counts.SkillsKnown || counts.Skills != 4 {
		t.Fatalf("expected known skills=4, got known=%t value=%d", counts.SkillsKnown, counts.Skills)
	}
	if !counts.PluginsKnown || counts.Plugins != 1 {
		t.Fatalf("expected known plugins=1, got known=%t value=%d", counts.PluginsKnown, counts.Plugins)
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

func TestCountEnvironmentFallsBackToUnknownWhenPluginStateUnavailable(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", home)

	counts := CountEnvironment(project)
	if counts.SkillsKnown {
		t.Fatalf("expected unknown skills")
	}
	if counts.PluginsKnown {
		t.Fatalf("expected unknown plugins")
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
