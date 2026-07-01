package assets

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"
)

const (
	maxImageDimension = 8192
	minImageDimension = 1
)

var (
	ErrFileTooLarge      = errors.New("uploaded file is too large")
	ErrUnsupportedType   = errors.New("unsupported image type")
	ErrInvalidDimensions = fmt.Errorf("unsupported image dimensions (supported: %dx%d to %dx%d)", minImageDimension, minImageDimension, maxImageDimension, maxImageDimension)
)

type UploadResult struct {
	ID       string
	Name     string
	Width    int
	Height   int
	FilePath string
}

func OptimizeUploadedMap(dataDir, fileName string, src io.Reader, maxUploadBytes int64) (*UploadResult, error) {
	if maxUploadBytes <= 0 {
		return nil, ErrFileTooLarge
	}

	raw, err := io.ReadAll(io.LimitReader(src, maxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read uploaded file: %w", err)
	}
	if int64(len(raw)) > maxUploadBytes {
		return nil, ErrFileTooLarge
	}

	contentType := http.DetectContentType(raw)
	if !isAllowedType(contentType) {
		return nil, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrUnsupportedType
	}
	if cfg.Width < minImageDimension || cfg.Height < minImageDimension || cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
		return nil, ErrInvalidDimensions
	}

	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode uploaded image: %w", err)
	}

	id, err := newMapID()
	if err != nil {
		return nil, fmt.Errorf("generate map id: %w", err)
	}

	mapsDir := filepath.Join(dataDir, "maps")
	if err := os.MkdirAll(mapsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create maps directory: %w", err)
	}

	outPath := filepath.Join(mapsDir, id+".webp")
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("create output map: %w", err)
	}
	defer outFile.Close()

	if err := nativewebp.Encode(outFile, decoded, &nativewebp.Options{CompressionLevel: nativewebp.BestCompression}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}

	name := cleanMapName(fileName)

	return &UploadResult{
		ID:       id,
		Name:     name,
		Width:    cfg.Width,
		Height:   cfg.Height,
		FilePath: outPath,
	}, nil
}

func cleanMapName(name string) string {
	base := strings.TrimSpace(filepath.Base(name))
	if base == "" || base == "." {
		return "uploaded-map"
	}
	return base
}

func isAllowedType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func newMapID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
