package diskcache

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

func Read(path string, maxBytes int64, target any) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	return err == nil && int64(len(data)) <= maxBytes && json.Unmarshal(data, target) == nil
}

// 用临时文件替换完整快照，让并发刷新的进程读到完整缓存。
func Write(path string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	file, err := os.CreateTemp(dir, ".cache-*.tmp")
	if err != nil {
		return
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		return
	}
	// 缓存可以重建，无需同步落盘。
	_ = os.Rename(file.Name(), path)
}
