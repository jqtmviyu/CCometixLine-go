package state

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

const (
	bridgeVersion = 1
	bridgeMaxAge  = 6 * time.Hour
	bridgeMaxSize = 1 << 20
)

var safeSessionID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ReadBridge(sessionID string, now time.Time) *protocol.Bridge {
	if !safeSessionID.MatchString(sessionID) {
		return nil
	}

	path := filepath.Join(config.BridgeDir(), sessionID+".json")
	for attempt := 0; attempt < 2; attempt++ {
		bridge, err := readBridgeFile(path)
		if err == nil {
			if validBridge(bridge, sessionID, now) {
				return bridge
			}
			return nil
		}
		if os.IsNotExist(err) {
			return nil
		}
		if attempt == 0 {
			time.Sleep(3 * time.Millisecond)
		}
	}
	return nil
}

func readBridgeFile(path string) (*protocol.Bridge, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, bridgeMaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > bridgeMaxSize {
		return nil, errInvalidBridge
	}

	var bridge protocol.Bridge
	if err := json.Unmarshal(data, &bridge); err != nil {
		return nil, err
	}
	return &bridge, nil
}

func validBridge(bridge *protocol.Bridge, sessionID string, now time.Time) bool {
	if bridge.Version != bridgeVersion || bridge.Session != sessionID || bridge.UpdatedAt <= 0 {
		return false
	}
	age := now.Sub(time.UnixMilli(bridge.UpdatedAt))
	return age >= -time.Minute && age <= bridgeMaxAge
}

var errInvalidBridge = errors.New("companion bridge exceeds size limit")
