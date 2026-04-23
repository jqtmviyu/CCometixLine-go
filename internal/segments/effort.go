package segments

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

var knownEfforts = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
	"xhigh":  true,
	"max":    true,
}

var unknownEffortPattern = regexp.MustCompile(`^[a-z0-9-]{2,20}$`)

const effortEnvKey = "CLAUDE_CODE_EFFORT_LEVEL"

type EffortSegment struct {
	Input protocol.InputData
}

func (s EffortSegment) Collect() *SegmentData {
	effort := formatEffort(resolveEffort(s.Input))

	return &SegmentData{
		Primary:   effort,
		Secondary: "",
		Metadata: map[string]string{
			"effort": effort,
		},
	}
}

func resolveEffort(input protocol.InputData) string {
	if effort, ok := resolveEnvEffort(); ok {
		return effort
	}

	if effort, ok := normalizeEffort(input.Effort); ok {
		return effort
	}

	if effort, ok := resolveSettingsEffort(input.Workspace.CurrentDir); ok {
		return effort
	}

	return ""
}

func normalizeEffort(value string) (string, bool) {
	if value == "" {
		return "", false
	}

	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "auto" {
		return "", true
	}
	if knownEfforts[normalized] {
		return normalized, true
	}
	if unknownEffortPattern.MatchString(normalized) && strings.ContainsAny(normalized, "abcdefghijklmnopqrstuvwxyz0123456789") {
		return normalized + "?", true
	}

	return "", false
}

func resolveEnvEffort() (string, bool) {
	return normalizeEffort(os.Getenv(effortEnvKey))
}

func resolveSettingsEffort(cwd string) (string, bool) {
	paths := []string{filepath.Join(config.ClaudeDir(), "settings.json")}
	if cwd != "" {
		paths = append(paths,
			filepath.Join(cwd, ".claude", "settings.json"),
			filepath.Join(cwd, ".claude", "settings.local.json"),
		)
	}

	for i := len(paths) - 1; i >= 0; i-- {
		if effort, ok := readEffortFromSettings(paths[i]); ok {
			return effort, true
		}
	}

	return "", false
}

func readEffortFromSettings(path string) (string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	var parsed struct {
		EffortLevel string `json:"effortLevel"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return "", false
	}

	return normalizeEffort(parsed.EffortLevel)
}

func formatEffort(value string) string {
	if value == "" {
		return "auto"
	}

	return value
}

