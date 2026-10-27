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
	info, ok := getGitInfo(s.Input.Workspace.CurrentDir)
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

func getGitInfo(workingDir string) (GitInfo, bool) {
	output, err := runGit(workingDir, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return GitInfo{}, false
	}
	return parseGitStatusV2(output)
}

func parseGitStatusV2(output string) (GitInfo, bool) {
	info := GitInfo{}
	branchFound := false
	dirty := false
	conflicts := false

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			branch := strings.TrimSpace(strings.TrimPrefix(line, "# branch.head "))
			if branch != "" && branch != "(unknown)" && branch != "(initial)" {
				if branch == "(detached)" {
					branch = "detached"
				}
				info.Branch = branch
				branchFound = true
			}
		case strings.HasPrefix(line, "# branch.oid "):
			if !branchFound {
				oid := strings.TrimSpace(strings.TrimPrefix(line, "# branch.oid "))
				if oid != "" && oid != "(initial)" {
					info.Branch = "detached"
					branchFound = true
				}
			}
		case strings.HasPrefix(line, "u "):
			conflicts = true
			dirty = true
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "), strings.HasPrefix(line, "? "):
			dirty = true
		}
	}

	if !branchFound {
		return GitInfo{}, false
	}
	info.Status = GitStatusClean
	if conflicts {
		info.Status = GitStatusConflicts
	} else if dirty {
		info.Status = GitStatusDirty
	}
	return info, true
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
