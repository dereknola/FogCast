package config

import (
	"os"
	"strconv"
)

const defaultMaxUploadMB = 50
const defaultMaskSize = 512

type Config struct {
	Addr           string
	DataDir        string
	StaticDir      string
	MaxUploadBytes int64
	MaskSize       int
}

func Load() Config {
	maxUploadMB := envInt("FOGCAST_MAX_UPLOAD_MB", defaultMaxUploadMB)
	maskSize := envInt("FOGCAST_MASK_SIZE", defaultMaskSize)

	return Config{
		Addr:           env("FOGCAST_ADDR", ":8080"),
		DataDir:        env("FOGCAST_DATA_DIR", "data"),
		StaticDir:      env("FOGCAST_STATIC_DIR", "static"),
		MaxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
		MaskSize:       maskSize,
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
