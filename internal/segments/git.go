package segments

import (
	"os/exec"
	"strings"

	"ccometixline-go/internal/protocol"
)

type GitInfo struct {
	Branch string
	Status GitStatus
}

type GitStatus string

const (
	GitStatusClean     GitStatus = "Clean"
	GitStatusDirty     GitStatus = "Dirty"
	GitStatusConflicts GitStatus = "Conflicts"
)

type GitSegment struct {
	Input protocol.InputData
}

func (s GitSegment) Collect() *SegmentData {
	info, ok := s.getGitInfo(s.Input.Workspace.CurrentDir)
	if !ok {
		return nil
	}

	metadata := map[string]string{
		"branch": info.Branch,
		"status": string(info.Status),
	}

	statusParts := []string{}
	switch info.Status {
	case GitStatusClean:
		statusParts = append(statusParts, "✓")
	case GitStatusDirty:
		statusParts = append(statusParts, "●")
	case GitStatusConflicts:
		statusParts = append(statusParts, "⚠")
	}

	return &SegmentData{
		Primary:   info.Branch,
		Secondary: strings.Join(statusParts, " "),
		Metadata:  metadata,
	}
}

func (s GitSegment) getGitInfo(workingDir string) (GitInfo, bool) {
	branch, ok := resolveGitHead(workingDir)
	if !ok {
		return GitInfo{}, false
	}
	if branch == "" {
		branch = "detached"
	}

	return GitInfo{
		Branch: branch,
		Status: s.getStatus(workingDir),
	}, true
}

func resolveGitHead(workingDir string) (string, bool) {
	if _, err := runGit(workingDir, "rev-parse", "--git-dir"); err != nil {
		return "", false
	}

	output, err := runGit(workingDir, "branch", "--show-current")
	if err != nil {
		return "detached", true
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		return "detached", true
	}
	return branch, true
}

func (s GitSegment) getStatus(workingDir string) GitStatus {
	output, err := runGit(workingDir, "status", "--porcelain")
	if err != nil {
		return GitStatusClean
	}

	statusText := strings.TrimSpace(output)
	if statusText == "" {
		return GitStatusClean
	}
	if strings.Contains(statusText, "UU") || strings.Contains(statusText, "AA") || strings.Contains(statusText, "DD") {
		return GitStatusConflicts
	}
	return GitStatusDirty
}

func runGit(workingDir string, args ...string) (string, error) {
	cmdArgs := append([]string{"--no-optional-locks"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
