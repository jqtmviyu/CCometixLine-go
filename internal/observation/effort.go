package observation

import (
	"regexp"
	"strings"
)

var knownEfforts = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
	"xhigh":  true,
	"max":    true,
}

var unknownEffortPattern = regexp.MustCompile(`^[a-z0-9-]{2,20}$`)

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
