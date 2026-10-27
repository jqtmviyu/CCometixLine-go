package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ccometixline-go/internal/config"
)

func TestReadBridgeMatchesSessionVersionAndFreshness(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	bridgeDir := config.BridgeDir()
	if err := os.MkdirAll(bridgeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	path := filepath.Join(bridgeDir, "session_1.json")
	if err := os.WriteFile(path, []byte(`{"v":1,"session":"session_1","updated_at":1700000000000,"turn":{"ttft_ms":820}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if bridge := ReadBridge("session_1", now); bridge == nil || bridge.Turn == nil || bridge.Turn.TTFTMS == nil || *bridge.Turn.TTFTMS != 820 {
		t.Fatalf("expected valid bridge, got %#v", bridge)
	}
	if bridge := ReadBridge("other", now); bridge != nil {
		t.Fatalf("expected other session to be ignored, got %#v", bridge)
	}
	if bridge := ReadBridge("../session_1", now); bridge != nil {
		t.Fatalf("expected unsafe session id to be ignored, got %#v", bridge)
	}
	if err := os.WriteFile(path, []byte(`{"v":2,"session":"session_1","updated_at":1700000000000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if bridge := ReadBridge("session_1", now); bridge != nil {
		t.Fatalf("expected unsupported version to be ignored, got %#v", bridge)
	}
	if err := os.WriteFile(path, []byte(`{"v":1,"session":"session_1","updated_at":1699970000000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if bridge := ReadBridge("session_1", now); bridge != nil {
		t.Fatalf("expected stale bridge to be ignored, got %#v", bridge)
	}
}
