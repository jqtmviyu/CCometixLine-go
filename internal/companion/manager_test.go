package companion

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeResponse struct {
	stdout string
	stderr string
	err    error
}

type fakeRunner struct {
	responses map[string][]fakeResponse
	calls     [][]string
}

func (r *fakeRunner) Run(_ context.Context, binary string, args []string) ([]byte, []byte, error) {
	call := append([]string{binary}, args...)
	r.calls = append(r.calls, call)
	key := strings.Join(call, " ")
	responses := r.responses[key]
	if len(responses) == 0 {
		return nil, nil, errors.New("unexpected command: " + key)
	}
	response := responses[0]
	r.responses[key] = responses[1:]
	return []byte(response.stdout), []byte(response.stderr), response.err
}

func TestParseArgs(t *testing.T) {
	for _, test := range []struct {
		args      []string
		operation string
		wantErr   bool
	}{
		{args: nil, operation: "help"},
		{args: []string{"help"}, operation: "help"},
		{args: []string{"install"}, operation: "install"},
		{args: []string{"status"}, operation: "status"},
		{args: []string{"uninstall"}, operation: "uninstall"},
		{args: []string{"install", "--source", "/repo/companion"}, wantErr: true},
		{args: []string{"status", "extra"}, wantErr: true},
		{args: []string{"remove"}, wantErr: true},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			operation, err := parseArgs(test.args)
			if (err != nil) != test.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && operation != test.operation {
				t.Fatalf("got operation=%q, want %q", operation, test.operation)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	for _, test := range []struct {
		version string
		want    bool
	}{
		{version: "2.1.287 (Claude Code)", want: true},
		{version: "2.1.309", want: true},
		{version: "2.1.286", want: false},
		{version: "2.0.999", want: false},
		{version: "unknown", want: false},
	} {
		if got := versionAtLeast(test.version, minimumVersion); got != test.want {
			t.Errorf("versionAtLeast(%q) = %v, want %v", test.version, got, test.want)
		}
	}
}

func TestEmbeddedMarketplaceWritesPrivateFilesAndContentVersion(t *testing.T) {
	version, err := embeddedPluginVersion()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(version, pluginBaseVersion+"+") || len(strings.TrimPrefix(version, pluginBaseVersion+"+")) != 8 {
		t.Fatalf("unexpected embedded plugin version %q", version)
	}
	if again, err := embeddedPluginVersion(); err != nil || again != version {
		t.Fatalf("embedded version is not stable: got %q, err %v", again, err)
	}

	root := filepath.Join(t.TempDir(), "mod")
	if err := writeEmbeddedMarketplace(root, version); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("marketplace permissions = %04o, want 0700", info.Mode().Perm())
	}

	for _, name := range embeddedFiles {
		path := filepath.Join(root, filepath.FromSlash(name))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("embedded file %s was not written: %v", name, err)
		}
		if name == "ccline-go/.claude-plugin/plugin.json" {
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var manifest map[string]any
			if err := json.Unmarshal(contents, &manifest); err != nil {
				t.Fatal(err)
			}
			if manifest["version"] != version {
				t.Fatalf("plugin version = %v, want %s", manifest["version"], version)
			}
			fileInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if fileInfo.Mode().Perm() != 0o600 {
				t.Fatalf("plugin manifest permissions = %04o, want 0600", fileInfo.Mode().Perm())
			}
		}
	}
}

func TestInstallUpdatesExistingPluginAndCreatesPrivateBridge(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	root := filepath.Join(claudeDir, "ccline", "mod")
	version, err := embeddedPluginVersion()
	if err != nil {
		t.Fatal(err)
	}
	installed := pluginList(t, version, true)
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":                                               {{stdout: "2.1.309 (Claude Code)"}},
		"claude plugin list --json":                                      {{stdout: installed}, {stdout: installed}},
		"claude plugin marketplace list --json":                          {{stdout: marketplaceList(t, root)}},
		"claude plugin marketplace update ccometixline-go":               {{stdout: "marketplace updated"}},
		"claude plugin update ccline-go@ccometixline-go --scope user -y": {{stdout: "plugin updated"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	if err := manager.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 6 {
		t.Fatalf("unexpected commands for update install: %#v", runner.calls)
	}
	info, err := os.Stat(filepath.Join(claudeDir, "ccline", "bridge"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("bridge permissions = %04o, want 0700", info.Mode().Perm())
	}
}

func TestInstallRegistersEmbeddedMarketplaceAndInstallsPlugin(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	root := filepath.Join(claudeDir, "ccline", "mod")
	version, err := embeddedPluginVersion()
	if err != nil {
		t.Fatal(err)
	}
	installed := pluginList(t, version, true)
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":                                             {{stdout: "2.1.309 (Claude Code)"}},
		"claude plugin list --json":                                    {{stdout: "[]"}, {stdout: installed}},
		"claude plugin marketplace list --json":                        {{stdout: "[]"}},
		"claude plugin marketplace add " + root + " --scope user":      {{stdout: "marketplace added"}},
		"claude plugin install ccline-go@ccometixline-go --scope user": {{stdout: "plugin installed"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	if err := manager.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantCalls := [][]string{
		{"claude", "--version"},
		{"claude", "plugin", "list", "--json"},
		{"claude", "plugin", "marketplace", "list", "--json"},
		{"claude", "plugin", "marketplace", "add", root, "--scope", "user"},
		{"claude", "plugin", "install", "ccline-go@ccometixline-go", "--scope", "user"},
		{"claude", "plugin", "list", "--json"},
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Fatalf("unexpected install commands:\n got %#v\nwant %#v", runner.calls, wantCalls)
	}
	if _, err := os.Stat(filepath.Join(root, "ccline-go", "hooks", "ccline.ts")); err != nil {
		t.Fatalf("embedded hook was not staged: %v", err)
	}
}

func TestInstallReenablesDisabledPluginAfterUpdate(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	root := filepath.Join(claudeDir, "ccline", "mod")
	version, err := embeddedPluginVersion()
	if err != nil {
		t.Fatal(err)
	}
	disabled := pluginList(t, "0.1.0", false)
	enabled := pluginList(t, version, true)
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":                                               {{stdout: "2.1.309"}},
		"claude plugin list --json":                                      {{stdout: disabled}, {stdout: enabled}},
		"claude plugin marketplace list --json":                          {{stdout: marketplaceList(t, root)}},
		"claude plugin marketplace update ccometixline-go":               {{stdout: "marketplace updated"}},
		"claude plugin update ccline-go@ccometixline-go --scope user -y": {{stdout: "plugin updated"}},
		"claude plugin enable ccline-go@ccometixline-go --scope user":    {{stdout: "plugin enabled"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	if err := manager.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[len(runner.calls)-2]; !reflect.DeepEqual(got, []string{"claude", "plugin", "enable", "ccline-go@ccometixline-go", "--scope", "user"}) {
		t.Fatalf("disabled plugin was not enabled after update: %#v", runner.calls)
	}
}

func TestInstallMigratesExistingLocalMarketplace(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	root := filepath.Join(claudeDir, "ccline", "mod")
	oldRoot := filepath.Join(t.TempDir(), "companion")
	version, err := embeddedPluginVersion()
	if err != nil {
		t.Fatal(err)
	}
	oldPlugin := pluginList(t, "0.1.0", true)
	updatedPlugin := pluginList(t, version, true)
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":                                               {{stdout: "2.1.309"}},
		"claude plugin list --json":                                      {{stdout: oldPlugin}, {stdout: updatedPlugin}},
		"claude plugin marketplace list --json":                          {{stdout: marketplaceList(t, oldRoot)}},
		"claude plugin marketplace remove ccometixline-go --scope user":  {{stdout: "marketplace removed"}},
		"claude plugin marketplace add " + root + " --scope user":        {{stdout: "marketplace added"}},
		"claude plugin update ccline-go@ccometixline-go --scope user -y": {{stdout: "plugin updated"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	if err := manager.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[3:5]; !reflect.DeepEqual(got, [][]string{
		{"claude", "plugin", "marketplace", "remove", "ccometixline-go", "--scope", "user"},
		{"claude", "plugin", "marketplace", "add", root, "--scope", "user"},
	}) {
		t.Fatalf("marketplace was not migrated to embedded files: %#v", got)
	}
}

func TestInstallDoesNotReplaceMarketplaceUsedByOtherPlugins(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	oldRoot := filepath.Join(t.TempDir(), "shared-marketplace")
	plugins := `[{"id":"ccline-go@ccometixline-go","version":"0.1.0","scope":"user","enabled":true},{"id":"other@ccometixline-go","version":"1.0.0","scope":"user","enabled":true}]`
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":                      {{stdout: "2.1.309"}},
		"claude plugin list --json":             {{stdout: plugins}},
		"claude plugin marketplace list --json": {{stdout: marketplaceList(t, oldRoot)}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	err := manager.install(context.Background())
	if err == nil || !strings.Contains(err.Error(), "other@ccometixline-go") {
		t.Fatalf("expected shared-marketplace conflict, got %v", err)
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call, " "), "marketplace remove") {
			t.Fatalf("shared marketplace was removed: %#v", runner.calls)
		}
	}
}

func TestInstallRefusesOldClaudeVersion(t *testing.T) {
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version": {{stdout: "2.1.200"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	err := manager.install(context.Background())
	if err == nil || !strings.Contains(err.Error(), minimumVersion) {
		t.Fatalf("expected minimum-version error, got %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("plugin commands ran on unsupported Claude Code: %#v", runner.calls)
	}
}

func TestInstallRejectsUnownedMarketplaceFromDifferentPath(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	registered := filepath.Join(t.TempDir(), "other-marketplace")
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":                      {{stdout: "2.1.309"}},
		"claude plugin list --json":             {{stdout: "[]"}},
		"claude plugin marketplace list --json": {{stdout: marketplaceList(t, registered)}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	err := manager.install(context.Background())
	if err == nil || !strings.Contains(err.Error(), registered) {
		t.Fatalf("expected existing-marketplace conflict, got %v", err)
	}
}

func TestStatusReportsPluginAndBridgeSnapshot(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	bridgeDir := filepath.Join(claudeDir, "ccline", "bridge")
	if err := os.MkdirAll(bridgeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bridgeDir, "session.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	version, err := embeddedPluginVersion()
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":          {{stdout: "2.1.309"}},
		"claude plugin list --json": {{stdout: pluginList(t, version, true)}},
	}}
	var output strings.Builder
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: &output, Stderr: io.Discard}
	if err := manager.status(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Claude Code: 2.1.309", "Mod: enabled (" + version + ", user scope)", "Bridge: 1 session snapshots"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("status output missing %q: %s", want, output.String())
		}
	}
}

func TestStatusReportsEmbeddedUpdate(t *testing.T) {
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude --version":          {{stdout: "2.1.309"}},
		"claude plugin list --json": {{stdout: pluginList(t, "0.1.0", true)}},
	}}
	var output strings.Builder
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: &output, Stderr: io.Discard}
	if err := manager.status(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "update available") {
		t.Fatalf("status did not report embedded update: %s", output.String())
	}
}

func TestUninstallRemovesPluginAndMarketplaceButKeepsBridge(t *testing.T) {
	claudeDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	bridgeDir := filepath.Join(claudeDir, "ccline", "bridge")
	if err := os.MkdirAll(bridgeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	bridgeFile := filepath.Join(bridgeDir, "session.json")
	if err := os.WriteFile(bridgeFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude plugin list --json":                                      {{stdout: pluginList(t, "0.1.0", true)}, {stdout: "[]"}},
		"claude plugin uninstall ccline-go@ccometixline-go --scope user": {{stdout: "plugin uninstalled"}},
		"claude plugin marketplace list --json":                          {{stdout: marketplaceList(t, filepath.Join(claudeDir, "ccline", "mod"))}},
		"claude plugin marketplace remove ccometixline-go --scope user":  {{stdout: "marketplace removed"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	if err := manager.uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bridgeFile); err != nil {
		t.Fatalf("uninstall deleted bridge data: %v", err)
	}
	if got := runner.calls[1]; !reflect.DeepEqual(got, []string{"claude", "plugin", "uninstall", "ccline-go@ccometixline-go", "--scope", "user"}) {
		t.Fatalf("unexpected uninstall command: %#v", got)
	}
}

func TestUninstallKeepsMarketplaceUsedByAnotherPlugin(t *testing.T) {
	remaining := `[{"id":"other@ccometixline-go","version":"1.0.0","scope":"user","enabled":true}]`
	runner := &fakeRunner{responses: map[string][]fakeResponse{
		"claude plugin list --json": {
			{stdout: pluginList(t, "0.1.0", true)},
			{stdout: remaining},
		},
		"claude plugin uninstall ccline-go@ccometixline-go --scope user": {{stdout: "plugin uninstalled"}},
	}}
	manager := Manager{Runner: runner, ClaudePath: "claude", Stdout: io.Discard, Stderr: io.Discard}
	if err := manager.uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call, " "), "marketplace remove") {
			t.Fatalf("shared marketplace was removed: %#v", runner.calls)
		}
	}
}

func pluginList(t *testing.T, version string, enabled bool) string {
	t.Helper()
	data, err := json.Marshal([]pluginRecord{{
		ID:      pluginName + "@" + marketplaceName,
		Version: version,
		Scope:   "user",
		Enabled: enabled,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func marketplaceList(t *testing.T, path string) string {
	t.Helper()
	data, err := json.Marshal([]marketplaceRecord{{Name: marketplaceName, Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
