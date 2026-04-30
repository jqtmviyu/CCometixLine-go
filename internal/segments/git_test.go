package segments

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ccometixline-go/internal/protocol"
)

func TestGitSegmentReturnsNilOutsideRepo(t *testing.T) {
	segment := GitSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: t.TempDir()}}}
	if data := segment.Collect(); data != nil {
		t.Fatal("expected nil outside git repo")
	}
}

func TestGitSegmentCollectsDirtyRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "tester")

	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", "file.txt")
	runGitCmd(t, dir, "commit", "-m", "init")

	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	segment := GitSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: dir}}}
	data := segment.Collect()
	if data == nil {
		t.Fatal("expected git data")
	}
	if data.Primary == "" {
		t.Fatal("expected branch name")
	}
	if !strings.Contains(data.Secondary, "●") {
		t.Fatalf("expected dirty marker, got %q", data.Secondary)
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}
