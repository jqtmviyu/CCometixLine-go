package config

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"time"

	"ccometixline-go/internal/diskcache"
)

const (
	environmentCacheVersion = 1
	environmentCacheTTL     = 60 * time.Second
	environmentCacheMaxSize = 4096
)

type environmentCacheRecord struct {
	Version   int               `json:"version"`
	UpdatedAt int64             `json:"updated_at"`
	Counts    EnvironmentCounts `json:"counts"`
}

func readEnvironmentCache(claudeDir string, cwd string, now time.Time, ttl time.Duration) (EnvironmentCounts, bool) {
	var record environmentCacheRecord
	if !diskcache.Read(environmentCachePath(claudeDir, cwd), environmentCacheMaxSize, &record) || record.Version != environmentCacheVersion {
		return EnvironmentCounts{}, false
	}
	age := now.Sub(time.UnixMilli(record.UpdatedAt))
	if age < 0 || age >= ttl || !validEnvironmentCounts(record.Counts) {
		return EnvironmentCounts{}, false
	}
	return record.Counts, true
}

func writeEnvironmentCache(claudeDir string, cwd string, counts EnvironmentCounts, now time.Time) {
	diskcache.Write(environmentCachePath(claudeDir, cwd), environmentCacheRecord{
		Version: environmentCacheVersion, UpdatedAt: now.UnixMilli(), Counts: counts,
	})
}

func environmentCachePath(claudeDir string, cwd string) string {
	scope := normalizeEnvironmentCachePath(claudeDir) + "\x00" + normalizeEnvironmentCachePath(cwd)
	sum := sha256.Sum256([]byte(scope))
	name := hex.EncodeToString(sum[:]) + ".json"
	return filepath.Join(CacheDir(), "environment", name)
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
