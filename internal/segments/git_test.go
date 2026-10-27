package segments

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ccometixline-go/internal/protocol"
)

func TestGitSegmentReturnsNilOutsideRepo(t *testing.T) {
	segment := GitSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: t.TempDir()}}}
	if data := segment.Collect(); data != nil {
		t.Fatal("expected nil outside git repo")
	}
}

func TestParseGitStatusV2(t *testing.T) {
	for _, tc := range []struct {
		name   string
		output string
		branch string
		status GitStatus
		ok     bool
	}{
		{
			name:   "clean branch",
			output: "# branch.oid abc123\n# branch.head main\n",
			branch: "main",
			status: GitStatusClean,
			ok:     true,
		},
		{
			name:   "staged and unstaged changes",
			output: "# branch.head feature\n1 MM N... 100644 100644 100644 abc abc file.txt\n",
			branch: "feature",
			status: GitStatusDirty,
			ok:     true,
		},
		{
			name:   "rename",
			output: "# branch.head main\n2 R. N... 100644 100644 100644 R100 old.txt\tnew.txt\n",
			branch: "main",
			status: GitStatusDirty,
			ok:     true,
		},
		{
			name:   "untracked",
			output: "# branch.head main\n? new.txt\n",
			branch: "main",
			status: GitStatusDirty,
			ok:     true,
		},
		{
			name:   "unmerged",
			output: "# branch.head main\nu UU N... 100644 100644 100644 100644 a b c d file.txt\n",
			branch: "main",
			status: GitStatusConflicts,
			ok:     true,
		},
		{
			name:   "detached head",
			output: "# branch.oid abc123\n# branch.head (detached)\n",
			branch: "detached",
			status: GitStatusClean,
			ok:     true,
		},
		{
			name:   "initial branch",
			output: "# branch.oid (initial)\n# branch.head main\n",
			branch: "main",
			status: GitStatusClean,
			ok:     true,
		},
		{
			name:   "missing branch headers",
			output: "1 .M N... 100644 100644 100644 abc abc file.txt\n",
			ok:     false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := parseGitStatusV2(tc.output)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v, info=%#v", ok, tc.ok, info)
			}
			if !tc.ok {
				return
			}
			if info.Branch != tc.branch || info.Status != tc.status {
				t.Fatalf("got %#v, want branch=%q status=%q", info, tc.branch, tc.status)
			}
		})
	}
}

func TestGitSegmentCollectsCleanRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "tester")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", "file.txt")
	runGitCmd(t, dir, "commit", "-m", "init")

	data := (GitSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: dir}}}).Collect()
	if data == nil || !strings.Contains(data.Secondary, "✓") {
		t.Fatalf("expected clean repo, got %#v", data)
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

func TestGitCacheRefreshesChangesAndSeparatesDirectories(t *testing.T) {
	t.Setenv("CCLINE_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "-b", "main")
	now := time.Now().Truncate(time.Second)
	ttl := 3 * time.Second
	first, ok := getCachedGitInfo(dir, ttl, now)
	if !ok || first.Branch != "main" || first.Status != GitStatusClean {
		t.Fatalf("initial status: %+v, %v", first, ok)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	cached, ok := getCachedGitInfo(dir, ttl, now.Add(time.Second))
	if !ok || cached != first {
		t.Fatalf("expected cached clean status: %+v", cached)
	}
	refreshed, ok := getCachedGitInfo(dir, ttl, now.Add(ttl))
	if !ok || refreshed.Status != GitStatusDirty {
		t.Fatalf("expected refreshed dirty status: %+v", refreshed)
	}
	runGitCmd(t, dir, "symbolic-ref", "HEAD", "refs/heads/feature")
	immediate, ok := getCachedGitInfo(dir, 0, now.Add(ttl))
	if !ok || immediate.Branch != "feature" {
		t.Fatalf("zero TTL should read current branch: %+v", immediate)
	}
	other := t.TempDir()
	runGitCmd(t, other, "init", "-b", "other")
	distinct, ok := getCachedGitInfo(other, ttl, now)
	if !ok || distinct.Branch != "other" || distinct.Status != GitStatusClean {
		t.Fatalf("mixed directory caches: %+v", distinct)
	}
}

func TestGitCacheRechecksNewRepositoryAfterTTL(t *testing.T) {
	t.Setenv("CCLINE_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)
	ttl := 3 * time.Second
	if _, ok := getCachedGitInfo(dir, ttl, now); ok {
		t.Fatal("expected no repository")
	}
	runGitCmd(t, dir, "init", "-b", "main")
	if _, ok := getCachedGitInfo(dir, ttl, now.Add(time.Second)); ok {
		t.Fatal("expected cached absence")
	}
	if info, ok := getCachedGitInfo(dir, ttl, now.Add(ttl)); !ok || info.Branch != "main" {
		t.Fatalf("repository should appear after expiry: %+v, %v", info, ok)
	}
}
