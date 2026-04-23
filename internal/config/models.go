package config

import (
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type ModelConfig struct {
	ModelEntries     []ModelEntry      `toml:"models"`
	ContextModifiers []ContextModifier `toml:"context_modifiers"`
}

type ModelEntry struct {
	Pattern      string `toml:"pattern"`
	DisplayName  string `toml:"display_name"`
	ContextLimit uint32 `toml:"context_limit"`
}

type ContextModifier struct {
	Pattern       string `toml:"pattern"`
	DisplaySuffix string `toml:"display_suffix"`
	ContextLimit  uint32 `toml:"context_limit"`
}

type builtinModelFamily struct {
	regex         *regexp.Regexp
	displayPrefix string
	contextLimit  uint32
}

func newBuiltinModelFamily(keyword string, displayPrefix string, contextLimit uint32) builtinModelFamily {
	pattern := `(?:(?P<pre_major>\d{1,2})(?:-(?P<pre_minor>\d{1,2}))?-` + keyword + `|` + keyword + `-(?P<post_major>\d{1,2})(?:-(?P<post_minor>\d{1,2}))?)(?:-\d{3,}|-[a-z]|\[|$)`
	return builtinModelFamily{
		regex:         regexp.MustCompile(pattern),
		displayPrefix: displayPrefix,
		contextLimit:  contextLimit,
	}
}

func (f builtinModelFamily) MatchModel(modelIDLower string) (string, bool) {
	matches := f.regex.FindStringSubmatch(modelIDLower)
	if matches == nil {
		return "", false
	}

	major := namedGroup(f.regex, matches, "post_major")
	if major == "" {
		major = namedGroup(f.regex, matches, "pre_major")
	}
	if major == "" {
		return "", false
	}

	minor := namedGroup(f.regex, matches, "post_minor")
	if minor == "" {
		minor = namedGroup(f.regex, matches, "pre_minor")
	}

	version := major
	if minor != "" {
		version += "." + minor
	}

	return f.displayPrefix + " " + version, true
}

func defaultModelConfig() ModelConfig {
	return ModelConfig{
		ModelEntries: []ModelEntry{
			{Pattern: "glm-4.5", DisplayName: "GLM-4.5", ContextLimit: 128000},
			{Pattern: "kimi-k2-turbo", DisplayName: "Kimi K2 Turbo", ContextLimit: 128000},
			{Pattern: "kimi-k2", DisplayName: "Kimi K2", ContextLimit: 128000},
			{Pattern: "qwen3-coder", DisplayName: "Qwen Coder", ContextLimit: 256000},
		},
		ContextModifiers: []ContextModifier{
			{Pattern: "[1m]", DisplaySuffix: " 1M", ContextLimit: 1000000},
		},
	}
}

func builtinFamilies() []builtinModelFamily {
	return []builtinModelFamily{
		newBuiltinModelFamily("sonnet", "Sonnet", 200000),
		newBuiltinModelFamily("opus", "Opus", 200000),
		newBuiltinModelFamily("haiku", "Haiku", 200000),
	}
}

func LoadModelConfig() ModelConfig {
	config := defaultModelConfig()
	path := ModelsPath()
	if _, err := os.Stat(path); err != nil {
		return config
	}

	var external ModelConfig
	if _, err := toml.DecodeFile(path, &external); err != nil {
		return config
	}

	config.ModelEntries = append(external.ModelEntries, config.ModelEntries...)
	config.ContextModifiers = append(external.ContextModifiers, config.ContextModifiers...)
	return config
}

func (m ModelConfig) GetContextLimit(modelID string) uint32 {
	_, limit, _ := m.resolve(modelID)
	return limit
}

func (m ModelConfig) GetDisplayName(modelID string) (string, bool) {
	display, _, _ := m.resolve(modelID)
	if display == "" {
		return "", false
	}
	return display, true
}

func (m ModelConfig) GetDisplaySuffix(modelID string) (string, bool) {
	_, _, suffix := m.resolve(modelID)
	if suffix == "" {
		return "", false
	}
	return suffix, true
}

func (m ModelConfig) resolve(modelID string) (string, uint32, string) {
	modelLower := strings.ToLower(modelID)

	baseName := ""
	baseLimit := uint32(0)
	for _, entry := range m.ModelEntries {
		if strings.Contains(modelLower, strings.ToLower(entry.Pattern)) {
			baseName = entry.DisplayName
			baseLimit = entry.ContextLimit
			break
		}
	}

	if baseName == "" {
		for _, family := range builtinFamilies() {
			if name, ok := family.MatchModel(modelLower); ok {
				baseName = name
				baseLimit = family.contextLimit
				break
			}
		}
	}

	modifierSuffix := ""
	modifierLimit := uint32(0)
	for _, modifier := range m.ContextModifiers {
		if strings.Contains(modelLower, strings.ToLower(modifier.Pattern)) {
			modifierSuffix = modifier.DisplaySuffix
			modifierLimit = modifier.ContextLimit
			break
		}
	}

	displayName := baseName
	if displayName != "" && modifierSuffix != "" {
		displayName += modifierSuffix
	}

	contextLimit := uint32(200000)
	if baseLimit > 0 {
		contextLimit = baseLimit
	}
	if modifierLimit > 0 {
		contextLimit = modifierLimit
	}

	return displayName, contextLimit, modifierSuffix
}

func namedGroup(re *regexp.Regexp, matches []string, name string) string {
	index := re.SubexpIndex(name)
	if index <= 0 || index >= len(matches) {
		return ""
	}
	return matches[index]
}
