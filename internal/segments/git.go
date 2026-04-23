package segments

import (
	"os/exec"
	"strings"

	"ccometixline-go/internal/protocol"
)

type GitInfo struct {
	Branch string
	Status GitStatus
	SHA    string
}

type GitStatus string

const (
	GitStatusClean     GitStatus = "Clean"
	GitStatusDirty     GitStatus = "Dirty"
	GitStatusConflicts GitStatus = "Conflicts"
)

type GitSegment struct {
	Input   protocol.InputData
	ShowSHA bool
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
	if info.SHA != "" {
		metadata["sha"] = info.SHA
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
	if info.SHA != "" {
		statusParts = append(statusParts, info.SHA)
	}

	return &SegmentData{
		Primary:   info.Branch,
		Secondary: strings.Join(statusParts, " "),
		Metadata:  metadata,
	}
}

func (s GitSegment) getGitInfo(workingDir string) (GitInfo, bool) {
	branch, sha, ok := resolveGitHead(workingDir)
	if !ok {
		return GitInfo{}, false
	}
	if branch == "" {
		branch = "detached"
	}

	info := GitInfo{
		Branch: branch,
		Status: s.getStatus(workingDir),
	}
	if s.ShowSHA {
		info.SHA = sha
	}
	return info, true
}

func resolveGitHead(workingDir string) (string, string, bool) {
	output, err := runGit(workingDir, "symbolic-ref", "--short", "HEAD")
	if err == nil {
		branch := strings.TrimSpace(output)
		if branch != "" {
			sha, _ := gitShortSHA(workingDir)
			return branch, sha, true
		}
	}

	sha, err := gitShortSHA(workingDir)
	if err != nil {
		return "", "", false
	}
	return "detached", sha, true
}

func gitShortSHA(workingDir string) (string, error) {
	output, err := runGit(workingDir, "rev-parse", "--short=7", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
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
