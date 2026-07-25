package segments

import (
	"os"
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

type EffortSegment struct {
	Input    protocol.InputData
	Settings config.SettingsSnapshot
}

func (s EffortSegment) Collect() *SegmentData {
	effort := formatEffort(resolveEffort(s.Input, s.Settings))

	return &SegmentData{
		Primary:   effort,
		Secondary: "",
		Metadata: map[string]string{
			"effort": effort,
		},
	}
}

func resolveEffort(input protocol.InputData, settings config.SettingsSnapshot) string {
	effort, ok := normalizeEffort(os.Getenv("CLAUDE_CODE_EFFORT_LEVEL"))
	if ok {
		return effort
	}

	effort, ok = normalizeEffort(payloadEffort(input))
	if ok {
		return effort
	}

	effort, ok = normalizeEffort(settings.ConfiguredEffortLevel())
	if ok {
		return effort
	}

	return ""
}

func payloadEffort(input protocol.InputData) string {
	if input.Effort == nil {
		return ""
	}

	return input.Effort.Level
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

func formatEffort(value string) string {
	if value == "" {
		return "auto"
	}

	return value
}

