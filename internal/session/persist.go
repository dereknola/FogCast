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
	Mask      []byte    `json:"mask,omitempty"`
}

func LoadActiveMap(dataDir string) (*MapState, error) {
	payload, err := loadPersistedState(dataDir)
	if err != nil {
		return nil, err
	}

	if payload.ActiveMap == nil {
		return nil, nil
	}

	return cloneMapState(payload.ActiveMap), nil
}

func SaveActiveMap(dataDir string, activeMap *MapState) error {
	payload, err := loadPersistedState(dataDir)
	if err != nil {
		return err
	}

	payload.ActiveMap = cloneMapState(activeMap)
	return writePersistedState(dataDir, payload)
}

func LoadMask(dataDir string, expectedLength int) ([]byte, error) {
	if expectedLength <= 0 {
		return nil, fmt.Errorf("invalid expected mask length: %d", expectedLength)
	}

	payload, err := loadPersistedState(dataDir)
	if err != nil {
		return nil, err
	}

	if len(payload.Mask) == 0 {
		return nil, nil
	}

	if len(payload.Mask) != expectedLength {
		return nil, nil
	}

	mask := make([]byte, len(payload.Mask))
	copy(mask, payload.Mask)
	return mask, nil
}

func SaveMask(dataDir string, mask []byte) error {
	payload, err := loadPersistedState(dataDir)
	if err != nil {
		return err
	}

	if len(mask) == 0 {
		payload.Mask = nil
	} else {
		payload.Mask = make([]byte, len(mask))
		copy(payload.Mask, mask)
	}

	return writePersistedState(dataDir, payload)
}

func loadPersistedState(dataDir string) (persistedState, error) {
	statePath := filepath.Join(dataDir, persistedStateFile)
	raw, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return persistedState{}, nil
		}
		return persistedState{}, fmt.Errorf("read persisted state: %w", err)
	}

	var payload persistedState
	if err := json.Unmarshal(raw, &payload); err != nil {
		return persistedState{}, fmt.Errorf("decode persisted state: %w", err)
	}

	return payload, nil
}

func writePersistedState(dataDir string, payload persistedState) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	statePath := filepath.Join(dataDir, persistedStateFile)
	tempPath := statePath + ".tmp"

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
