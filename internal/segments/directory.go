package segments

import (
	"path/filepath"
	"strings"

	"ccometixline-go/internal/protocol"
)

type DirectorySegment struct {
	Input protocol.InputData
}

func (s DirectorySegment) Collect() *SegmentData {
	return &SegmentData{
		Primary:   extractDirectoryName(s.Input.Workspace.CurrentDir),
		Secondary: "",
		Metadata: map[string]string{
			"full_path": s.Input.Workspace.CurrentDir,
		},
	}
}

func extractDirectoryName(path string) string {
	name := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	if name == "." || name == "/" || name == "" {
		return "root"
	}
	return name
}
