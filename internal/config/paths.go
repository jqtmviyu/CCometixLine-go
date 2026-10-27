package config

import (
	"os"
	"path/filepath"
	"strings"
)

func ClaudeDir() string {
	envPath := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR"))
	if envPath != "" {
		home, err := os.UserHomeDir()
		if err == nil {
			if envPath == "~" {
				return home
			}
			if strings.HasPrefix(envPath, "~/") || strings.HasPrefix(envPath, "~\\") {
				return filepath.Join(home, envPath[2:])
			}
		}
		return filepath.Clean(envPath)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}

	return filepath.Join(home, ".claude")
}

func CclineDir() string {
	return filepath.Join(ClaudeDir(), "ccline")
}

func ModelsPath() string {
	return filepath.Join(CclineDir(), "models.toml")
}

func BridgeDir() string {
	path := strings.TrimSpace(os.Getenv("CCLINE_BRIDGE_DIR"))
	if path == "" {
		return filepath.Join(CclineDir(), "bridge")
	}
	if home, err := os.UserHomeDir(); err == nil {
		if path == "~" {
			return home
		}
		if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
			return filepath.Join(home, path[2:])
		}
	}
	return filepath.Clean(path)
}
