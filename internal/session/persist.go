package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const persistedStateFile = "state.json"

type persistedState struct {
	ActiveMap *MapState `json:"activeMap"`
}

func LoadActiveMap(dataDir string) (*MapState, error) {
	statePath := filepath.Join(dataDir, persistedStateFile)
	raw, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read persisted state: %w", err)
	}

	var payload persistedState
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode persisted state: %w", err)
	}

	if payload.ActiveMap == nil {
		return nil, nil
	}

	return cloneMapState(payload.ActiveMap), nil
}

func SaveActiveMap(dataDir string, activeMap *MapState) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	statePath := filepath.Join(dataDir, persistedStateFile)
	tempPath := statePath + ".tmp"

	payload := persistedState{ActiveMap: cloneMapState(activeMap)}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode persisted state: %w", err)
	}

	if err := os.WriteFile(tempPath, raw, 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tempPath, statePath); err != nil {
		return fmt.Errorf("persist state: %w", err)
	}

	return nil
}
