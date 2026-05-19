package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type EnvironmentCounts struct {
	MemoryFiles int
	MCPs        int
	Skills      int
	Plugins     int
}

type SettingsSnapshot struct {
	counts         EnvironmentCounts
	configuredLevel string
}

type parsedSettings struct {
	EnabledPlugins        map[string]bool `json:"enabledPlugins"`
	EnabledMCPJSONServers []any           `json:"enabledMcpjsonServers"`
	EffortLevel           string          `json:"effortLevel"`
}

func LoadSettingsSnapshot(cwd string) SettingsSnapshot {
	claudeDir := ClaudeDir()
	projectClaudeDir := filepath.Join(cwd, ".claude")
	projectOverlapsUser := samePath(projectClaudeDir, claudeDir)
	settingsList := collectSettingsList(claudeDir, projectClaudeDir, projectOverlapsUser)

	memoryFiles := countMemoryFiles(claudeDir, cwd, projectClaudeDir, projectOverlapsUser)
	mcpNames := collectMCPNames(settingsList)
	pluginNames := collectPluginNames(settingsList)
	skills := len(collectSkillNames(claudeDir, cwd))

	return SettingsSnapshot{
		counts: EnvironmentCounts{
			MemoryFiles: memoryFiles,
			MCPs:        len(mcpNames),
			Skills:      skills,
			Plugins:     len(pluginNames),
		},
		configuredLevel: configuredEffortLevel(settingsList),
	}
}

func CountEnvironment(cwd string) EnvironmentCounts {
	return LoadSettingsSnapshot(cwd).EnvironmentCounts()
}

func ConfiguredEffortLevel(cwd string) string {
	return LoadSettingsSnapshot(cwd).ConfiguredEffortLevel()
}

func (s SettingsSnapshot) EnvironmentCounts() EnvironmentCounts {
	return s.counts
}

func (s SettingsSnapshot) ConfiguredEffortLevel() string {
	return s.configuredLevel
}

func readSettings(path string) parsedSettings {
	content, err := os.ReadFile(path)
	if err != nil {
		return parsedSettings{}
	}

	var parsed parsedSettings
	if err := json.Unmarshal(content, &parsed); err != nil {
		return parsedSettings{}
	}
	return parsed
}

func collectSettingsList(claudeDir string, projectClaudeDir string, projectOverlapsUser bool) []parsedSettings {
	claudeSettingsPath := filepath.Join(claudeDir, "settings.json")
	projectSettingsPath := filepath.Join(projectClaudeDir, "settings.json")
	projectLocalSettingsPath := filepath.Join(projectClaudeDir, "settings.local.json")

	result := []parsedSettings{readSettings(claudeSettingsPath)}
	if !projectOverlapsUser {
		result = append(result, readSettings(projectSettingsPath))
	}
	result = append(result, readSettings(projectLocalSettingsPath))
	return result
}

func configuredEffortLevel(settingsList []parsedSettings) string {
	for i := len(settingsList) - 1; i >= 0; i-- {
		value := strings.TrimSpace(settingsList[i].EffortLevel)
		if value != "" {
			return value
		}
	}

	return ""
}

func countMemoryFiles(claudeDir string, cwd string, projectClaudeDir string, projectOverlapsUser bool) int {
	userClaudePath := filepath.Join(claudeDir, "CLAUDE.md")
	count := 0
	count += fileCount(userClaudePath)
	count += markdownFileCount(filepath.Join(claudeDir, "rules"))
	count += distinctFileCount(filepath.Join(cwd, "CLAUDE.md"), userClaudePath)
	count += fileCount(filepath.Join(cwd, "CLAUDE.local.md"))
	if !projectOverlapsUser {
		count += fileCount(filepath.Join(projectClaudeDir, "CLAUDE.md"))
		count += markdownFileCount(filepath.Join(projectClaudeDir, "rules"))
	}
	count += fileCount(filepath.Join(projectClaudeDir, "CLAUDE.local.md"))
	return count
}

func collectMCPNames(settingsList []parsedSettings) map[string]struct{} {
	return mergeNameSets(settingsList, func(settings parsedSettings) map[string]struct{} {
		return stringSetFromAnySlice(settings.EnabledMCPJSONServers)
	})
}

func collectPluginNames(settingsList []parsedSettings) map[string]struct{} {
	return mergeNameSets(settingsList, func(settings parsedSettings) map[string]struct{} {
		return enabledPluginSet(settings.EnabledPlugins)
	})
}

func collectSkillNames(claudeDir string, cwd string) map[string]struct{} {
	return mergePathNameSets([]string{
		filepath.Join(claudeDir, "skills"),
		filepath.Join(cwd, ".claude", "skills"),
	}, directChildDirNames)
}

func mergeNameSets(settingsList []parsedSettings, names func(parsedSettings) map[string]struct{}) map[string]struct{} {
	result := map[string]struct{}{}
	for _, settings := range settingsList {
		for name := range names(settings) {
			result[name] = struct{}{}
		}
	}
	return result
}

func mergePathNameSets(paths []string, names func(string) map[string]struct{}) map[string]struct{} {
	result := map[string]struct{}{}
	for _, path := range paths {
		for name := range names(path) {
			result[name] = struct{}{}
		}
	}
	return result
}

func enabledPluginSet(values map[string]bool) map[string]struct{} {
	result := map[string]struct{}{}
	for name, enabled := range values {
		if enabled {
			result[name] = struct{}{}
		}
	}
	return result
}

func stringSetFromAnySlice(values []any) map[string]struct{} {
	result := map[string]struct{}{}
	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			continue
		}
		result[name] = struct{}{}
	}
	return result
}

func markdownFileCount(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			count += markdownFileCount(path)
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			count++
		}
	}
	return count
}

func directChildDirNames(dir string) map[string]struct{} {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return map[string]struct{}{}
	}

	result := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() {
			result[entry.Name()] = struct{}{}
		}
	}
	return result
}

func fileCount(path string) int {
	if isFile(path) {
		return 1
	}
	return 0
}

func distinctFileCount(path string, other string) int {
	if !isFile(path) {
		return 0
	}
	if samePath(path, other) {
		return 0
	}
	return 1
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func samePath(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftResolved := resolvedPath(left)
	rightResolved := resolvedPath(right)
	return leftResolved != "" && rightResolved != "" && leftResolved == rightResolved
}

func resolvedPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = resolved
	}
	abs = filepath.Clean(abs)
	return strings.ToLower(abs)
}
