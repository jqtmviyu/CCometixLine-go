package observation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadResolvesEffortSources(t *testing.T) {
	for _, tc := range []struct{ name, env, payload, user, project, local, want string }{
		{name: "environment", env: "HIGH", payload: "max", user: "low", project: "medium", want: "high"},
		{name: "environment auto", env: "auto", payload: "max", local: "high", want: ""},
		{name: "payload", payload: "medium", local: "high", want: "medium"},
		{name: "payload auto", payload: "auto", user: "high", want: ""},
		{name: "invalid environment falls back", env: "@@", payload: "max", user: "low", want: "max"},
		{name: "user", user: "low", want: "low"},
		{name: "project overrides user", user: "low", project: "high", want: "high"},
		{name: "local overrides project", project: "medium", local: "xhigh", want: "xhigh"},
		{name: "unknown payload", payload: "super-max", want: "super-max?"},
		{name: "unknown environment", env: "future-level", payload: "high", want: "future-level?"},
		{name: "missing", want: ""},
		{name: "invalid payload falls back", payload: "@@", local: "high", want: "high"},
		{name: "invalid settings", local: "@@", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, cwd := t.TempDir(), t.TempDir()
			setClaudeConfigDir(t, root)
			t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", tc.env)
			writeJSON(t, filepath.Join(root, "settings.json"), map[string]string{"effortLevel": tc.user})
			writeJSON(t, filepath.Join(cwd, ".claude", "settings.json"), map[string]string{"effortLevel": tc.project})
			writeJSON(t, filepath.Join(cwd, ".claude", "settings.local.json"), map[string]string{"effortLevel": tc.local})
			got := Read(Request{CWD: cwd, Effort: true, InputEffort: tc.payload})
			if got.Effort != tc.want || got.Environment != (EnvironmentCounts{}) {
				t.Fatalf("snapshot=%+v, want effort %q", got, tc.want)
			}
			if _, err := os.Stat(filepath.Join(root, "ccline", "cache", "environment")); !os.IsNotExist(err) {
				t.Fatalf("effort-only read created environment cache: %v", err)
			}
		})
	}
}
