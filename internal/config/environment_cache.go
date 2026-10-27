package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	environmentCacheVersion = 1
	environmentCacheTTL     = 10 * time.Second
	environmentCacheMaxSize = 4096
)

type environmentCacheRecord struct {
	Version   int               `json:"version"`
	UpdatedAt int64             `json:"updated_at"`
	Counts    EnvironmentCounts `json:"counts"`
}

func readEnvironmentCache(claudeDir string, cwd string, now time.Time) (EnvironmentCounts, bool) {
	file, err := os.Open(environmentCachePath(claudeDir, cwd))
	if err != nil {
		return EnvironmentCounts{}, false
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, environmentCacheMaxSize+1))
	if err != nil || len(data) > environmentCacheMaxSize {
		return EnvironmentCounts{}, false
	}

	var record environmentCacheRecord
	if err := json.Unmarshal(data, &record); err != nil || record.Version != environmentCacheVersion {
		return EnvironmentCounts{}, false
	}
	age := now.Sub(time.UnixMilli(record.UpdatedAt))
	if age < 0 || age > environmentCacheTTL || !validEnvironmentCounts(record.Counts) {
		return EnvironmentCounts{}, false
	}
	return record.Counts, true
}

func writeEnvironmentCache(claudeDir string, cwd string, counts EnvironmentCounts, now time.Time) {
	path := environmentCachePath(claudeDir, cwd)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}

	file, err := os.CreateTemp(dir, ".environment-*.tmp")
	if err != nil {
		return
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return
	}
	record := environmentCacheRecord{
		Version:   environmentCacheVersion,
		UpdatedAt: now.UnixMilli(),
		Counts:    counts,
	}
	if err := json.NewEncoder(file).Encode(record); err != nil {
		file.Close()
		return
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		return
	}
	_ = os.Rename(tempPath, path)
}

func environmentCachePath(claudeDir string, cwd string) string {
	scope := normalizeEnvironmentCachePath(claudeDir) + "\x00" + normalizeEnvironmentCachePath(cwd)
	sum := sha256.Sum256([]byte(scope))
	name := hex.EncodeToString(sum[:]) + ".json"
	return filepath.Join(BridgeDir(), "environment", name)
}

func normalizeEnvironmentCachePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = resolved
	}
	return filepath.Clean(absolute)
}

func validEnvironmentCounts(counts EnvironmentCounts) bool {
	return counts.MemoryFiles >= 0 && counts.MCPs >= 0 && counts.Skills >= 0 && counts.Plugins >= 0
}
