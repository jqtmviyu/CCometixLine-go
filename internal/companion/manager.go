package companion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	companionassets "ccometixline-go/companion"
	"ccometixline-go/internal/config"
)

const (
	pluginName        = "ccline-go"
	marketplaceName   = "ccometixline-go"
	pluginBaseVersion = "0.1.0"
	minimumVersion    = "2.1.287"
	modUsage          = "usage: ccline mod [install | status | uninstall]\n"
)

var embeddedFiles = []string{
	".claude-plugin/marketplace.json",
	"ccline-go/.claude-plugin/plugin.json",
	"ccline-go/hooks/hooks.json",
	"ccline-go/hooks/ccline.ts",
}

var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

type Runner interface {
	Run(context.Context, string, []string) ([]byte, []byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, binary string, args []string) ([]byte, []byte, error) {
	command := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return []byte(stdout.String()), []byte(stderr.String()), err
}

type Manager struct {
	Runner     Runner
	ClaudePath string
	Stdout     io.Writer
	Stderr     io.Writer
}

type pluginRecord struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Scope       string `json:"scope"`
	Enabled     bool   `json:"enabled"`
	InstallPath string `json:"installPath"`
}

type marketplaceRecord struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	InstallLocation string `json:"installLocation"`
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	operation, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if operation == "help" {
		_, _ = io.WriteString(stdout, modUsage)
		fmt.Fprintln(stdout, "install embeds and updates the user-scope mod; status shows plugin and bridge state; uninstall keeps bridge data")
		return 0
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(stderr, "Claude Code CLI `claude` was not found in PATH")
		return 1
	}
	manager := Manager{
		Runner:     execRunner{},
		ClaudePath: claudePath,
		Stdout:     stdout,
		Stderr:     stderr,
	}

	if err := manager.run(context.Background(), operation); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseArgs(args []string) (string, error) {
	if len(args) == 0 || len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		return "help", nil
	}
	if len(args) != 1 {
		return "", errors.New(strings.TrimSpace(modUsage))
	}

	switch args[0] {
	case "install", "status", "uninstall":
		return args[0], nil
	default:
		return "", errors.New(strings.TrimSpace(modUsage))
	}
}

func (m Manager) run(ctx context.Context, operation string) error {
	if m.Runner == nil {
		return errors.New("Claude command runner is unavailable")
	}
	switch operation {
	case "help":
		_, _ = io.WriteString(m.Stdout, modUsage)
		fmt.Fprintln(m.Stdout, "install embeds and updates the user-scope mod; status shows plugin and bridge state; uninstall keeps bridge data")
		return nil
	case "install":
		return m.install(ctx)
	case "status":
		return m.status(ctx)
	case "uninstall":
		return m.uninstall(ctx)
	default:
		return errors.New("unknown mod command")
	}
}

func (m Manager) install(ctx context.Context) error {
	claudeVersion, err := m.claudeVersion(ctx)
	if err != nil {
		return err
	}
	if !versionAtLeast(claudeVersion, minimumVersion) {
		return fmt.Errorf("mod requires Claude Code %s or newer; found %s", minimumVersion, claudeVersion)
	}

	version, err := embeddedPluginVersion()
	if err != nil {
		return err
	}
	marketplaceDir := filepath.Join(config.CclineDir(), "mod")
	if err := writeEmbeddedMarketplace(marketplaceDir, version); err != nil {
		return fmt.Errorf("write embedded mod files: %w", err)
	}
	plugins, err := m.plugins(ctx)
	if err != nil {
		return err
	}
	installed := findPlugin(plugins)
	marketplaces, err := m.marketplaces(ctx)
	if err != nil {
		return err
	}
	if err := m.ensureMarketplace(ctx, findMarketplace(marketplaces), plugins, installed, marketplaceDir); err != nil {
		return err
	}
	if err := ensureBridgeDirectory(); err != nil {
		return fmt.Errorf("prepare bridge directory: %w", err)
	}
	wasInstalled := installed != nil

	if installed == nil {
		if err := m.invoke(ctx, "plugin", "install", pluginName+"@"+marketplaceName, "--scope", "user"); err != nil {
			return err
		}
	} else if err := m.invoke(ctx, "plugin", "update", installed.ID, "--scope", "user", "-y"); err != nil {
		return err
	}
	if installed != nil && !installed.Enabled {
		if err := m.invoke(ctx, "plugin", "enable", installed.ID, "--scope", "user"); err != nil {
			return err
		}
	}

	installedPlugins, err := m.plugins(ctx)
	if err != nil {
		return err
	}
	installed = findPlugin(installedPlugins)
	if installed == nil || !installed.Enabled {
		return errors.New("Claude Code did not report ccline-go as installed and enabled")
	}
	if wasInstalled {
		fmt.Fprintln(m.Stdout, "ccline mod updated")
	} else {
		fmt.Fprintln(m.Stdout, "ccline mod installed")
	}
	fmt.Fprintln(m.Stdout, "run /reload-plugins in the current Claude Code session to apply the change")
	return nil
}

func (m Manager) ensureMarketplace(ctx context.Context, marketplace *marketplaceRecord, plugins []pluginRecord, installed *pluginRecord, root string) error {
	if marketplace == nil {
		return m.invoke(ctx, "plugin", "marketplace", "add", root, "--scope", "user")
	}

	registeredPath := marketplace.Path
	if registeredPath == "" {
		registeredPath = marketplace.InstallLocation
	}
	if registeredPath == "" {
		return fmt.Errorf("marketplace %q has no source path in Claude Code; remove it before installing the embedded mod", marketplaceName)
	}
	if samePath(registeredPath, root) {
		return m.invoke(ctx, "plugin", "marketplace", "update", marketplaceName)
	}
	if installed == nil {
		return fmt.Errorf("marketplace %q is already registered from %q; remove it before installing the embedded mod", marketplaceName, registeredPath)
	}
	for _, plugin := range plugins {
		if plugin.Scope == "user" && plugin.ID != installed.ID && strings.HasSuffix(plugin.ID, "@"+marketplaceName) {
			return fmt.Errorf("marketplace %q is also used by %s; move that plugin before updating the embedded mod", marketplaceName, plugin.ID)
		}
	}
	if err := m.invoke(ctx, "plugin", "marketplace", "remove", marketplaceName, "--scope", "user"); err != nil {
		return err
	}
	return m.invoke(ctx, "plugin", "marketplace", "add", root, "--scope", "user")
}

func (m Manager) status(ctx context.Context) error {
	claudeVersion, versionErr := m.claudeVersion(ctx)
	plugins, err := m.plugins(ctx)
	if err != nil {
		return err
	}
	if versionErr != nil {
		fmt.Fprintf(m.Stdout, "Claude Code: version unavailable (%v)\n", versionErr)
	} else {
		fmt.Fprintf(m.Stdout, "Claude Code: %s\n", claudeVersion)
	}

	currentVersion, err := embeddedPluginVersion()
	if err != nil {
		return err
	}
	if plugin := findPlugin(plugins); plugin != nil {
		state := "disabled"
		if plugin.Enabled {
			state = "enabled"
		}
		if plugin.Version != currentVersion {
			fmt.Fprintf(m.Stdout, "Mod: %s, update available (%s → %s, %s scope)\n", state, plugin.Version, currentVersion, plugin.Scope)
		} else {
			fmt.Fprintf(m.Stdout, "Mod: %s (%s, %s scope)\n", state, plugin.Version, plugin.Scope)
		}
	} else {
		fmt.Fprintln(m.Stdout, "Mod: not installed for user scope")
	}
	printBridgeStatus(m.Stdout, config.BridgeDir())
	return nil
}

func (m Manager) uninstall(ctx context.Context) error {
	plugins, err := m.plugins(ctx)
	if err != nil {
		return err
	}
	installed := findPlugin(plugins)
	if installed != nil {
		if err := m.invoke(ctx, "plugin", "uninstall", installed.ID, "--scope", "user"); err != nil {
			return err
		}
	}

	remaining, err := m.plugins(ctx)
	if err != nil {
		return err
	}
	marketplaceInUse := false
	for _, plugin := range remaining {
		if plugin.Scope == "user" && strings.HasSuffix(plugin.ID, "@"+marketplaceName) {
			marketplaceInUse = true
			break
		}
	}
	if !marketplaceInUse {
		marketplaces, err := m.marketplaces(ctx)
		if err != nil {
			return err
		}
		if findMarketplace(marketplaces) != nil {
			if err := m.invoke(ctx, "plugin", "marketplace", "remove", marketplaceName, "--scope", "user"); err != nil {
				return err
			}
		}
	}

	if installed == nil {
		fmt.Fprintln(m.Stdout, "ccline mod was not installed; nothing to uninstall")
	} else {
		fmt.Fprintln(m.Stdout, "ccline mod uninstalled")
	}
	fmt.Fprintf(m.Stdout, "bridge data was kept in %s\n", config.BridgeDir())
	fmt.Fprintln(m.Stdout, "run /reload-plugins in the current Claude Code session to apply the change")
	return nil
}

func (m Manager) claudeVersion(ctx context.Context) (string, error) {
	stdout, _, err := m.Runner.Run(ctx, m.ClaudePath, []string{"--version"})
	if err != nil {
		return "", fmt.Errorf("run claude --version: %w", err)
	}
	match := versionPattern.FindStringSubmatch(string(stdout))
	if match == nil {
		return "", fmt.Errorf("could not parse Claude Code version from %q", strings.TrimSpace(string(stdout)))
	}
	return fmt.Sprintf("%s.%s.%s", match[1], match[2], match[3]), nil
}

func (m Manager) plugins(ctx context.Context) ([]pluginRecord, error) {
	stdout, stderr, err := m.Runner.Run(ctx, m.ClaudePath, []string{"plugin", "list", "--json"})
	if err != nil {
		return nil, commandError("claude plugin list", stderr, err)
	}
	var plugins []pluginRecord
	if err := json.Unmarshal(stdout, &plugins); err != nil {
		return nil, fmt.Errorf("decode Claude plugin list: %w", err)
	}
	return plugins, nil
}

func (m Manager) marketplaces(ctx context.Context) ([]marketplaceRecord, error) {
	stdout, stderr, err := m.Runner.Run(ctx, m.ClaudePath, []string{"plugin", "marketplace", "list", "--json"})
	if err != nil {
		return nil, commandError("claude plugin marketplace list", stderr, err)
	}
	var marketplaces []marketplaceRecord
	if err := json.Unmarshal(stdout, &marketplaces); err != nil {
		return nil, fmt.Errorf("decode Claude marketplace list: %w", err)
	}
	return marketplaces, nil
}

func (m Manager) invoke(ctx context.Context, args ...string) error {
	stdout, stderr, err := m.Runner.Run(ctx, m.ClaudePath, args)
	if len(stdout) > 0 {
		_, _ = m.Stdout.Write(stdout)
		if stdout[len(stdout)-1] != '\n' {
			fmt.Fprintln(m.Stdout)
		}
	}
	if len(stderr) > 0 {
		_, _ = m.Stderr.Write(stderr)
	}
	if err != nil {
		return commandError("claude "+strings.Join(args, " "), stderr, err)
	}
	return nil
}

func findPlugin(plugins []pluginRecord) *pluginRecord {
	for i := range plugins {
		if plugins[i].ID == pluginName+"@"+marketplaceName && plugins[i].Scope == "user" {
			return &plugins[i]
		}
	}
	return nil
}

func findMarketplace(marketplaces []marketplaceRecord) *marketplaceRecord {
	for i := range marketplaces {
		if marketplaces[i].Name == marketplaceName {
			return &marketplaces[i]
		}
	}
	return nil
}

func embeddedPluginVersion() (string, error) {
	hash := fnv.New32a()
	for _, name := range embeddedFiles {
		contents, err := companionassets.Files.ReadFile(name)
		if err != nil {
			return "", fmt.Errorf("read embedded mod file %s: %w", name, err)
		}
		_, _ = fmt.Fprintf(hash, "%s\x00", name)
		_, _ = hash.Write(contents)
	}
	return fmt.Sprintf("%s+%08x", pluginBaseVersion, hash.Sum32()), nil
}

func writeEmbeddedMarketplace(root string, version string) error {
	if err := ensurePrivateDirectory(root); err != nil {
		return err
	}
	for _, name := range embeddedFiles {
		contents, err := companionassets.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read embedded mod file %s: %w", name, err)
		}
		if name == "ccline-go/.claude-plugin/plugin.json" {
			var manifest map[string]any
			if err := json.Unmarshal(contents, &manifest); err != nil {
				return fmt.Errorf("decode embedded plugin manifest: %w", err)
			}
			manifest["version"] = version
			contents, err = json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				return fmt.Errorf("encode embedded plugin manifest: %w", err)
			}
			contents = append(contents, '\n')
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := writePrivateFile(path, contents); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

func writePrivateFile(path string, contents []byte) error {
	dir := filepath.Dir(path)
	if err := ensurePrivateDirectory(dir); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".ccline-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}

func ensureBridgeDirectory() error {
	path := config.BridgeDir()
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}

func printBridgeStatus(writer io.Writer, path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(writer, "Bridge: not created (%s)\n", path)
			return
		}
		fmt.Fprintf(writer, "Bridge: unavailable (%v)\n", err)
		return
	}
	count := 0
	var newest time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		count++
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	if count == 0 {
		fmt.Fprintf(writer, "Bridge: waiting for session events (%s)\n", path)
		return
	}
	fmt.Fprintf(writer, "Bridge: %d session snapshots; newest %s\n", count, newest.Format(time.RFC3339))
}

func versionAtLeast(version string, minimum string) bool {
	current := versionPattern.FindStringSubmatch(version)
	required := versionPattern.FindStringSubmatch(minimum)
	if current == nil || required == nil {
		return false
	}
	for i := 1; i <= 3; i++ {
		var got, want int
		_, _ = fmt.Sscanf(current[i], "%d", &got)
		_, _ = fmt.Sscanf(required[i], "%d", &want)
		if got != want {
			return got > want
		}
	}
	return true
}

func samePath(left string, right string) bool {
	leftAbs, err := filepath.Abs(left)
	if err != nil {
		return false
	}
	rightAbs, err := filepath.Abs(right)
	if err != nil {
		return false
	}
	leftAbs = filepath.Clean(leftAbs)
	rightAbs = filepath.Clean(rightAbs)
	if leftResolved, err := filepath.EvalSymlinks(leftAbs); err == nil {
		leftAbs = leftResolved
	}
	if rightResolved, err := filepath.EvalSymlinks(rightAbs); err == nil {
		rightAbs = rightResolved
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(leftAbs, rightAbs)
	}
	return leftAbs == rightAbs
}

func commandError(command string, stderr []byte, err error) error {
	message := strings.TrimSpace(string(stderr))
	if message == "" {
		return fmt.Errorf("%s: %w", command, err)
	}
	return fmt.Errorf("%s: %w: %s", command, err, message)
}
