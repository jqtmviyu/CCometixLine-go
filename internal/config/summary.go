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

type parsedSettings struct {
	MCPServers             map[string]json.RawMessage `json:"mcpServers"`
	EnabledPlugins         map[string]bool            `json:"enabledPlugins"`
	DisabledMCPServers     []any                      `json:"disabledMcpServers"`
	DisabledMCPJSONServers []any                      `json:"disabledMcpjsonServers"`
}

type pluginInstall struct {
	InstallPath string `json:"installPath"`
}

type parsedPluginIndex struct {
	Plugins map[string][]pluginInstall `json:"plugins"`
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

	userSettings := readSettingsFile(claudeSettingsPath)
	userClaudeJSON := readSettingsFile(claudeJSONPath)
	projectSettings := parsedSettings{}
	if !projectOverlapsUser {
		projectSettings = readSettingsFile(projectSettingsPath)
	}
	projectLocalSettings := readSettingsFile(projectLocalSettingsPath)
	pluginIndex, pluginIndexKnown := readPluginIndex(installedPluginsPath)

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

	userMCPs := namesFromRawMap(userSettings.MCPServers)
	for name := range namesFromRawMap(userClaudeJSON.MCPServers) {
		userMCPs[name] = struct{}{}
	}
	for name := range valuesToNameSet(userClaudeJSON.DisabledMCPServers) {
		delete(userMCPs, name)
	}

	projectMCPs := map[string]struct{}{}
	if !projectOverlapsUser {
		for name := range namesFromRawMap(projectSettings.MCPServers) {
			projectMCPs[name] = struct{}{}
		}
	}
	for name := range namesFromRawMap(projectLocalSettings.MCPServers) {
		projectMCPs[name] = struct{}{}
	}
	mcpJSONServers := getMCPServerNames(filepath.Join(cwd, ".mcp.json"))
	for name := range valuesToNameSet(projectLocalSettings.DisabledMCPJSONServers) {
		delete(mcpJSONServers, name)
	}
	for name := range mcpJSONServers {
		projectMCPs[name] = struct{}{}
	}

	enabledPlugins, enabledPluginsKnown := enabledPluginNames(userSettings)
	skills, skillsKnown := countEnabledSkills(cwd, claudeDir, enabledPlugins, enabledPluginsKnown, pluginIndex, pluginIndexKnown)
	plugins, pluginsKnown := countEnabledPlugins(enabledPlugins, enabledPluginsKnown, pluginIndex, pluginIndexKnown)

	return EnvironmentCounts{
		MemoryFiles:  memoryFiles,
		MCPs:         len(userMCPs) + len(projectMCPs),
		Skills:       skills,
		Plugins:      plugins,
		SkillsKnown:  skillsKnown,
		PluginsKnown: pluginsKnown,
	}
}

func countEnabledSkills(cwd string, claudeDir string, enabledPlugins []string, enabledPluginsKnown bool, pluginIndex parsedPluginIndex, pluginIndexKnown bool) (int, bool) {
	count := 0
	count += countMarkdownFiles(filepath.Join(claudeDir, "skills"))
	count += countMarkdownFiles(filepath.Join(cwd, ".claude", "skills"))

	if enabledPluginsKnown {
		count += countPluginCommands(enabledPlugins, pluginIndex)
		return count, true
	}

	if count > 0 {
		return count, true
	}

	if pluginIndexKnown {
		return 0, false
	}

	return 0, false
}

func countEnabledPlugins(enabledPlugins []string, enabledPluginsKnown bool, pluginIndex parsedPluginIndex, pluginIndexKnown bool) (int, bool) {
	if enabledPluginsKnown {
		return len(enabledPlugins), true
	}

	if pluginIndexKnown {
		return len(pluginIndex.Plugins), true
	}

	return 0, false
}

func enabledPluginNames(settings parsedSettings) ([]string, bool) {
	if settings.EnabledPlugins == nil {
		return nil, false
	}

	enabled := []string{}
	for name, on := range settings.EnabledPlugins {
		if on {
			enabled = append(enabled, name)
		}
	}
	sort.Strings(enabled)
	return enabled, true
}

func countPluginCommands(enabledPlugins []string, pluginIndex parsedPluginIndex) int {
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

func findManifestInIndex(index parsedPluginIndex, plugin string) string {
	for _, install := range index.Plugins[plugin] {
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

func readSettingsFile(path string) parsedSettings {
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

func readPluginIndex(path string) (parsedPluginIndex, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return parsedPluginIndex{}, false
	}

	var parsed parsedPluginIndex
	if err := json.Unmarshal(content, &parsed); err != nil {
		return parsedPluginIndex{}, false
	}
	if parsed.Plugins == nil {
		return parsedPluginIndex{}, false
	}
	return parsed, true
}

func namesFromRawMap(values map[string]json.RawMessage) map[string]struct{} {
	result := map[string]struct{}{}
	for name := range values {
		result[name] = struct{}{}
	}
	return result
}

func valuesToNameSet(values []any) map[string]struct{} {
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
	settings := readSettingsFile(path)
	if key == disabledMcpServersKey {
		return valuesToNameSet(settings.DisabledMCPServers)
	}
	if key == disabledMcpJSONServersKey {
		return valuesToNameSet(settings.DisabledMCPJSONServers)
	}
	return map[string]struct{}{}
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
