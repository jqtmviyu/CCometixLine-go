package config

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type EnvironmentCounts struct {
	MemoryFiles int `json:"memory_files"`
	MCPs        int `json:"mcps"`
	Skills      int `json:"skills"`
	Plugins     int `json:"plugins"`
}

type SettingsSnapshot struct {
	counts          EnvironmentCounts
	configuredLevel string
}

type parsedSettings struct {
	EnabledPlugins        map[string]bool `json:"enabledPlugins"`
	EnabledMCPJSONServers []any           `json:"enabledMcpjsonServers"`
	EffortLevel           string          `json:"effortLevel"`
}

type settingsSnapshotOptions struct {
	includeEnvironment bool
	includeEffort      bool
	useCache           bool
	cacheTTL           time.Duration
}

func LoadSettingsSnapshot(cwd string) SettingsSnapshot {
	return loadSettingsSnapshotAt(cwd, settingsSnapshotOptions{includeEnvironment: true, includeEffort: true}, time.Now())
}

func LoadSettingsSnapshotForSegments(cwd string, segmentIDs []SegmentID) SettingsSnapshot {
	return LoadSettingsSnapshotWithCache(cwd, segmentIDs, environmentCacheTTL)
}

func LoadSettingsSnapshotWithCache(cwd string, segmentIDs []SegmentID, ttl time.Duration) SettingsSnapshot {
	options := settingsSnapshotOptions{useCache: ttl > 0, cacheTTL: ttl}
	for _, id := range segmentIDs {
		switch id {
		case SegmentEnvironment:
			options.includeEnvironment = true
		case SegmentEffort:
			options.includeEffort = true
		}
	}
	return loadSettingsSnapshotAt(cwd, options, time.Now())
}

func loadSettingsSnapshotAt(cwd string, options settingsSnapshotOptions, now time.Time) SettingsSnapshot {
	if !options.includeEnvironment && !options.includeEffort {
		return SettingsSnapshot{}
	}

	claudeDir := ClaudeDir()
	projectClaudeDir := filepath.Join(cwd, ".claude")
	projectOverlapsUser := samePath(projectClaudeDir, claudeDir)

	var counts EnvironmentCounts
	cacheHit := false
	if options.includeEnvironment && options.useCache {
		ttl := environmentCacheTTL
		if options.cacheTTL > 0 {
			ttl = options.cacheTTL
		}
		counts, cacheHit = readEnvironmentCache(claudeDir, cwd, now, ttl)
	}

	var settingsList []parsedSettings
	if options.includeEffort || (options.includeEnvironment && !cacheHit) {
		settingsList = collectSettingsList(claudeDir, projectClaudeDir, projectOverlapsUser)
	}

	if options.includeEnvironment && !cacheHit {
		counts = collectEnvironmentCounts(claudeDir, cwd, projectClaudeDir, projectOverlapsUser, settingsList)
		if options.useCache {
			writeEnvironmentCache(claudeDir, cwd, counts, now)
		}
	}

	snapshot := SettingsSnapshot{counts: counts}
	if options.includeEffort {
		snapshot.configuredLevel = configuredEffortLevel(settingsList)
	}
	return snapshot
}

func collectEnvironmentCounts(claudeDir string, cwd string, projectClaudeDir string, projectOverlapsUser bool, settingsList []parsedSettings) EnvironmentCounts {
	return EnvironmentCounts{
		MemoryFiles: countMemoryFiles(claudeDir, cwd, projectClaudeDir, projectOverlapsUser),
		MCPs:        len(collectMCPNames(settingsList)),
		Skills:      len(collectSkillNames(claudeDir, cwd)),
		Plugins:     len(collectPluginNames(settingsList)),
	}
}

func CountEnvironment(cwd string) EnvironmentCounts {
	return LoadSettingsSnapshotForSegments(cwd, []SegmentID{SegmentEnvironment}).EnvironmentCounts()
}

func ConfiguredEffortLevel(cwd string) string {
	return LoadSettingsSnapshotForSegments(cwd, []SegmentID{SegmentEffort}).ConfiguredEffortLevel()
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
	return mergePathNameSets(uniquePaths([]string{
		filepath.Join(claudeDir, "skills"),
		filepath.Join(cwd, ".claude", "skills"),
	}), callableSkillNames)
}

func uniquePaths(paths []string) []string {
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		duplicate := false
		for _, existing := range unique {
			if samePath(path, existing) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			unique = append(unique, path)
		}
	}
	return unique
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

func callableSkillNames(dir string) map[string]struct{} {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return map[string]struct{}{}
	}

	result := map[string]struct{}{}
	for _, entry := range entries {
		isDir := entry.IsDir()
		// 跟随 skill 子目录的软链接，让共享安装的 skill 也参与统计。
		if entry.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(filepath.Join(dir, entry.Name()))
			isDir = err == nil && info.IsDir()
		}
		if isDir && skillAllowsModelInvocation(filepath.Join(dir, entry.Name(), "SKILL.md")) {
			result[entry.Name()] = struct{}{}
		}
	}
	return result
}

func skillAllowsModelInvocation(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if strings.TrimSpace(strings.TrimPrefix(line, "\uFEFF")) != "---" {
		return err == nil || err == io.EOF
	}

	// 只读取 YAML 头部，正文中的配置示例不参与调用权限判断。
	var header strings.Builder
	for err == nil {
		line, err = reader.ReadString('\n')
		if marker := strings.TrimSpace(line); marker == "---" || marker == "..." {
			var metadata struct {
				Disabled bool `yaml:"disable-model-invocation"`
			}
			if yaml.Unmarshal([]byte(header.String()), &metadata) != nil {
				return false
			}
			return !metadata.Disabled
		}
		header.WriteString(line)
	}
	return false
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
