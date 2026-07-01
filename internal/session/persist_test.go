package session

import (
	"bytes"
	"testing"
)

func TestSaveAndLoadMaskSnapshot(t *testing.T) {
	dataDir := t.TempDir()
	mask := bytes.Repeat([]byte{123}, 512*512)

	if err := SaveMask(dataDir, mask); err != nil {
		t.Fatalf("save mask: %v", err)
	}

	loaded, err := LoadMask(dataDir, len(mask))
	if err != nil {
		t.Fatalf("load mask: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected loaded mask snapshot")
	}
	if !bytes.Equal(loaded, mask) {
		t.Fatalf("loaded mask mismatch")
	}
}

func TestSaveActiveMapPreservesMaskSnapshot(t *testing.T) {
	dataDir := t.TempDir()
	mask := bytes.Repeat([]byte{255}, 512*512)
	if err := SaveMask(dataDir, mask); err != nil {
		t.Fatalf("save initial mask: %v", err)
	}

	active := &MapState{ID: "abc", Name: "map.webp", Width: 100, Height: 80, URL: "/assets/maps/abc.webp"}
	if err := SaveActiveMap(dataDir, active); err != nil {
		t.Fatalf("save active map: %v", err)
	}

	loadedMask, err := LoadMask(dataDir, len(mask))
	if err != nil {
		t.Fatalf("load mask after active map save: %v", err)
	}
	if !bytes.Equal(loadedMask, mask) {
		t.Fatalf("expected mask snapshot to be preserved")
	}
}

func TestSaveMaskPreservesActiveMap(t *testing.T) {
	dataDir := t.TempDir()
	active := &MapState{ID: "def", Name: "dungeon.webp", Width: 200, Height: 150, URL: "/assets/maps/def.webp"}
	if err := SaveActiveMap(dataDir, active); err != nil {
		t.Fatalf("save active map: %v", err)
	}

	mask := bytes.Repeat([]byte{1}, 512*512)
	if err := SaveMask(dataDir, mask); err != nil {
		t.Fatalf("save mask snapshot: %v", err)
	}

	loadedMap, err := LoadActiveMap(dataDir)
	if err != nil {
		t.Fatalf("load active map after mask save: %v", err)
	}
	if loadedMap == nil {
		t.Fatalf("expected active map to be preserved")
	}
	if *loadedMap != *active {
		t.Fatalf("loaded active map mismatch: got %+v want %+v", *loadedMap, *active)
	}
}

func TestLoadMaskLengthMismatchReturnsNil(t *testing.T) {
	dataDir := t.TempDir()
	mask := bytes.Repeat([]byte{5}, 512*512)
	if err := SaveMask(dataDir, mask); err != nil {
		t.Fatalf("save mask snapshot: %v", err)
	}

	loaded, err := LoadMask(dataDir, 256*256)
	if err != nil {
		t.Fatalf("load mask with mismatched length: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil mask for length mismatch")
	}
}
