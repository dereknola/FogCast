package assets

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestOptimizeUploadedMapSuccess(t *testing.T) {
	t.Parallel()

	payload := mustEncodePNG(t, sampleImage(24, 16))
	dataDir := t.TempDir()

	result, err := OptimizeUploadedMap(dataDir, " encounter-map.png ", bytes.NewReader(payload), int64(len(payload))+1024)
	if err != nil {
		t.Fatalf("optimize uploaded map: %v", err)
	}

	if result.ID == "" {
		t.Fatalf("expected non-empty map id")
	}
	if result.Name != "encounter-map.png" {
		t.Fatalf("expected cleaned file name, got %q", result.Name)
	}
	if result.Width != 24 || result.Height != 16 {
		t.Fatalf("expected image dimensions 24x16, got %dx%d", result.Width, result.Height)
	}

	expectedPath := filepath.Join(dataDir, "maps", result.ID+".webp")
	if result.FilePath != expectedPath {
		t.Fatalf("expected file path %q, got %q", expectedPath, result.FilePath)
	}

	info, err := os.Stat(result.FilePath)
	if err != nil {
		t.Fatalf("stat encoded output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected encoded output to be non-empty")
	}
}

func TestOptimizeUploadedMapRejectsTooLarge(t *testing.T) {
	t.Parallel()

	payload := mustEncodePNG(t, sampleImage(20, 20))
	_, err := OptimizeUploadedMap(t.TempDir(), "map.png", bytes.NewReader(payload), int64(len(payload)-1))
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestOptimizeUploadedMapRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	_, err := OptimizeUploadedMap(t.TempDir(), "notes.txt", bytes.NewReader([]byte("not an image")), 1024)
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestCleanMapName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim and base", in: "  maps/dragon-cave.png  ", want: "dragon-cave.png"},
		{name: "empty", in: "", want: "uploaded-map"},
		{name: "dot", in: ".", want: "uploaded-map"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cleanMapName(tc.in)
			if got != tc.want {
				t.Fatalf("cleanMapName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsAllowedType(t *testing.T) {
	t.Parallel()

	if !isAllowedType("image/png") || !isAllowedType("image/jpeg") || !isAllowedType("image/webp") || !isAllowedType("image/gif") {
		t.Fatalf("expected known image types to be allowed")
	}
	if isAllowedType("text/plain") {
		t.Fatalf("expected text/plain to be rejected")
	}
}

func TestNewMapID(t *testing.T) {
	t.Parallel()

	id, err := newMapID()
	if err != nil {
		t.Fatalf("newMapID failed: %v", err)
	}
	if len(id) != 16 {
		t.Fatalf("expected 16-char hex id, got %q", id)
	}

	another, err := newMapID()
	if err != nil {
		t.Fatalf("newMapID second call failed: %v", err)
	}
	if id == another {
		t.Fatalf("expected two ids to differ, got %q", id)
	}
}

func sampleImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 7), B: 140, A: 255})
		}
	}
	return img
}

func mustEncodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
