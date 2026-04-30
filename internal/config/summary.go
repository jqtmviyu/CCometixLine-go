package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type EnvironmentCounts struct {
	MemoryFiles  int
	MCPs         int
	Skills       int
	Plugins      int
	SkillsKnown  bool
	PluginsKnown bool
}

type disabledMcpKey string

const (
	disabledMcpServersKey     disabledMcpKey = "disabledMcpServers"
	disabledMcpJSONServersKey disabledMcpKey = "disabledMcpjsonServers"
)

func CountEnvironment(cwd string) EnvironmentCounts {
	claudeDir := ClaudeDir()
	claudeJSONPath := claudeDir + ".json"
	claudeSettingsPath := filepath.Join(claudeDir, "settings.json")
	installedPluginsPath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	projectClaudeDir := filepath.Join(cwd, ".claude")
	projectSettingsPath := filepath.Join(projectClaudeDir, "settings.json")
	projectLocalSettingsPath := filepath.Join(projectClaudeDir, "settings.local.json")
	projectOverlapsUser := pathsReferToSameLocation(projectClaudeDir, claudeDir)

	userClaudePath := filepath.Join(claudeDir, "CLAUDE.md")
	memoryFiles := 0
	memoryFiles += countIfFile(userClaudePath)
	memoryFiles += countMarkdownFiles(filepath.Join(claudeDir, "rules"))
	memoryFiles += countIfDistinctFile(filepath.Join(cwd, "CLAUDE.md"), userClaudePath)
	memoryFiles += countIfFile(filepath.Join(cwd, "CLAUDE.local.md"))
	if !projectOverlapsUser {
		memoryFiles += countIfFile(filepath.Join(projectClaudeDir, "CLAUDE.md"))
		memoryFiles += countMarkdownFiles(filepath.Join(projectClaudeDir, "rules"))
	}
	memoryFiles += countIfFile(filepath.Join(projectClaudeDir, "CLAUDE.local.md"))

	userMCPs := getMCPServerNames(claudeSettingsPath)
	for name := range getMCPServerNames(claudeJSONPath) {
		userMCPs[name] = struct{}{}
	}
	for name := range getDisabledMCPServers(claudeJSONPath, disabledMcpServersKey) {
		delete(userMCPs, name)
	}

	projectMCPs := map[string]struct{}{}
	if !projectOverlapsUser {
		for name := range getMCPServerNames(projectSettingsPath) {
			projectMCPs[name] = struct{}{}
		}
	}
	for name := range getMCPServerNames(projectLocalSettingsPath) {
		projectMCPs[name] = struct{}{}
	}
	mcpJSONServers := getMCPServerNames(filepath.Join(cwd, ".mcp.json"))
	for name := range getDisabledMCPServers(projectLocalSettingsPath, disabledMcpJSONServersKey) {
		delete(mcpJSONServers, name)
	}
	for name := range mcpJSONServers {
		projectMCPs[name] = struct{}{}
	}

	skills, skillsKnown := countEnabledSkills(cwd, claudeDir, claudeSettingsPath, installedPluginsPath)
	plugins, pluginsKnown := countEnabledPlugins(claudeSettingsPath, installedPluginsPath)

	return EnvironmentCounts{
		MemoryFiles:  memoryFiles,
		MCPs:         len(userMCPs) + len(projectMCPs),
		Skills:       skills,
		Plugins:      plugins,
		SkillsKnown:  skillsKnown,
		PluginsKnown: pluginsKnown,
	}
}

func countEnabledSkills(cwd string, claudeDir string, settingsPath string, installedPluginsPath string) (int, bool) {
	count := 0
	count += countMarkdownFiles(filepath.Join(claudeDir, "skills"))
	count += countMarkdownFiles(filepath.Join(cwd, ".claude", "skills"))

	enabledPlugins, ok := readEnabledPlugins(settingsPath)
	if ok {
		count += countPluginCommands(enabledPlugins, installedPluginsPath)
		return count, true
	}

	if count > 0 {
		return count, true
	}

	return 0, false
}

func countEnabledPlugins(settingsPath string, installedPluginsPath string) (int, bool) {
	enabledPlugins, ok := readEnabledPlugins(settingsPath)
	if ok {
		return len(enabledPlugins), true
	}

	count, found := countInstalledPlugins(installedPluginsPath)
	if found {
		return count, true
	}

	return 0, false
}

func readEnabledPlugins(path string) ([]string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var parsed struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, false
	}
	if parsed.EnabledPlugins == nil {
		return nil, false
	}

	enabled := []string{}
	for name, on := range parsed.EnabledPlugins {
		if on {
			enabled = append(enabled, name)
		}
	}
	sort.Strings(enabled)
	return enabled, true
}

func countInstalledPlugins(path string) (int, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	var parsed struct {
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return 0, false
	}
	if parsed.Plugins == nil {
		return 0, false
	}

	return len(parsed.Plugins), true
}

func countPluginCommands(enabledPlugins []string, installedPluginsPath string) int {
	pluginIndex := loadPluginIndex(installedPluginsPath)
	count := 0
	for _, plugin := range enabledPlugins {
		manifestPath := findManifestInIndex(pluginIndex, plugin)
		if manifestPath == "" {
			continue
		}
		count += countCommandsFromManifest(manifestPath)
	}
	return count
}

func loadPluginIndex(installedPluginsPath string) map[string][]struct {
	InstallPath string `json:"installPath"`
} {
	content, err := os.ReadFile(installedPluginsPath)
	if err != nil {
		return nil
	}

	var parsed struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil
	}

	return parsed.Plugins
}

func findManifestInIndex(index map[string][]struct {
	InstallPath string `json:"installPath"`
}, plugin string) string {
	for _, install := range index[plugin] {
		manifestPath := filepath.Join(install.InstallPath, ".claude-plugin", "plugin.json")
		if fileExists(manifestPath) {
			return manifestPath
		}
	}
	return ""
}

func countCommandsFromManifest(path string) int {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	var parsed struct {
		Commands []string `json:"commands"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return 0
	}

	count := 0
	for _, command := range parsed.Commands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		count++
	}
	return count
}

func getMCPServerNames(path string) map[string]struct{} {
	content, err := os.ReadFile(path)
	if err != nil {
		return map[string]struct{}{}
	}

	var parsed struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return map[string]struct{}{}
	}

	result := map[string]struct{}{}
	for name := range parsed.MCPServers {
		result[name] = struct{}{}
	}
	return result
}

func getDisabledMCPServers(path string, key disabledMcpKey) map[string]struct{} {
	content, err := os.ReadFile(path)
	if err != nil {
		return map[string]struct{}{}
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(content, &parsed); err != nil {
		return map[string]struct{}{}
	}

	raw, ok := parsed[string(key)]
	if !ok {
		return map[string]struct{}{}
	}

	var values []any
	if err := json.Unmarshal(raw, &values); err != nil {
		return map[string]struct{}{}
	}

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

func countMarkdownFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			count += countMarkdownFiles(path)
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			count++
		}
	}
	return count
}

func countIfFile(path string) int {
	if fileExists(path) {
		return 1
	}
	return 0
}

func countIfDistinctFile(path string, other string) int {
	if !fileExists(path) {
		return 0
	}
	if pathsReferToSameLocation(path, other) {
		return 0
	}
	return 1
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathsReferToSameLocation(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftResolved := resolvePath(left)
	rightResolved := resolvePath(right)
	return leftResolved != "" && rightResolved != "" && leftResolved == rightResolved
}

func resolvePath(path string) string {
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
