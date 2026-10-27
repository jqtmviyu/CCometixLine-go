package segments

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/diskcache"
)

type gitCacheRecord struct {
	Version   int     `json:"version"`
	UpdatedAt int64   `json:"updated_at"`
	Found     bool    `json:"found"`
	Info      GitInfo `json:"info"`
}

func gitCachePath(cwd string) string {
	absolute, err := filepath.Abs(cwd)
	if err != nil {
		absolute = cwd
	}
	// Git 环境变量可以让同一目录指向不同仓库或索引，分别保存缓存。
	scope := []string{absolute}
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		scope = append(scope, os.Getenv(key))
	}
	sum := sha256.Sum256([]byte(strings.Join(scope, "\x00")))
	return filepath.Join(config.CacheDir(), "git", hex.EncodeToString(sum[:])+".json")
}

func getCachedGitInfo(cwd string, ttl time.Duration, now time.Time) (GitInfo, bool) {
	if ttl <= 0 {
		return getGitInfo(cwd)
	}
	path := gitCachePath(cwd)
	var record gitCacheRecord
	if diskcache.Read(path, 4096, &record) && record.Version == 1 {
		age := now.Sub(time.UnixMilli(record.UpdatedAt))
		if age >= 0 && age < ttl {
			return record.Info, record.Found
		}
	}
	info, found := getGitInfo(cwd)
	diskcache.Write(path, gitCacheRecord{Version: 1, UpdatedAt: now.UnixMilli(), Found: found, Info: info})
	return info, found
}
