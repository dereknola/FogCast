package config

import (
	"os"
	"strconv"
)

const defaultMaxUploadMB = 50
const defaultMaskSize = 512
const (
	defaultAddr          = ":8080"
	defaultDataDir       = "data"
	defaultStaticDir     = "static"
	containerDataDir     = "/data"
	containerStaticDir   = "/app/static"
	containerFlagEnvName = "FOGCAST_CONTAINER"
)

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
	runningInContainer := isContainerRuntime()

	return Config{
		Addr:           env("FOGCAST_ADDR", defaultAddr),
		DataDir:        env("FOGCAST_DATA_DIR", fallbackDataDir(runningInContainer)),
		StaticDir:      env("FOGCAST_STATIC_DIR", fallbackStaticDir(runningInContainer)),
		MaxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
		MaskSize:       maskSize,
	}
}

func fallbackDataDir(runningInContainer bool) string {
	if runningInContainer {
		return containerDataDir
	}

	return defaultDataDir
}

func fallbackStaticDir(runningInContainer bool) string {
	if runningInContainer {
		return containerStaticDir
	}

	return defaultStaticDir
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

func isContainerRuntime() bool {
	if value, exists := os.LookupEnv(containerFlagEnvName); exists {
		normalized := value
		switch normalized {
		case "1", "true", "TRUE", "yes", "YES", "on", "ON":
			return true
		case "0", "false", "FALSE", "no", "NO", "off", "OFF":
			return false
		default:
			return true
		}
	}

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}
