package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("FOGCAST_ADDR", "")
	t.Setenv("FOGCAST_DATA_DIR", "")
	t.Setenv("FOGCAST_STATIC_DIR", "")
	t.Setenv("FOGCAST_MAX_UPLOAD_MB", "")

	cfg := Load()

	if cfg.Addr != ":8080" {
		t.Fatalf("expected default addr :8080, got %q", cfg.Addr)
	}
	if cfg.DataDir != "data" {
		t.Fatalf("expected default data dir data, got %q", cfg.DataDir)
	}
	if cfg.StaticDir != "static" {
		t.Fatalf("expected default static dir static, got %q", cfg.StaticDir)
	}
	if cfg.MaxUploadBytes != 50*1024*1024 {
		t.Fatalf("expected default max upload bytes %d, got %d", 50*1024*1024, cfg.MaxUploadBytes)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("FOGCAST_ADDR", "127.0.0.1:9090")
	t.Setenv("FOGCAST_DATA_DIR", "/tmp/fogcast-data")
	t.Setenv("FOGCAST_STATIC_DIR", "/tmp/fogcast-static")
	t.Setenv("FOGCAST_MAX_UPLOAD_MB", "64")

	cfg := Load()

	if cfg.Addr != "127.0.0.1:9090" {
		t.Fatalf("expected env addr, got %q", cfg.Addr)
	}
	if cfg.DataDir != "/tmp/fogcast-data" {
		t.Fatalf("expected env data dir, got %q", cfg.DataDir)
	}
	if cfg.StaticDir != "/tmp/fogcast-static" {
		t.Fatalf("expected env static dir, got %q", cfg.StaticDir)
	}
	if cfg.MaxUploadBytes != 64*1024*1024 {
		t.Fatalf("expected max upload bytes %d, got %d", 64*1024*1024, cfg.MaxUploadBytes)
	}
}

func TestEnv(t *testing.T) {
	t.Setenv("FOGCAST_TEST_KEY", "")
	if got := env("FOGCAST_TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	t.Setenv("FOGCAST_TEST_KEY", "value")
	if got := env("FOGCAST_TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "missing", value: "", want: 7},
		{name: "invalid", value: "abc", want: 7},
		{name: "zero", value: "0", want: 7},
		{name: "negative", value: "-3", want: 7},
		{name: "valid", value: "42", want: 42},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FOGCAST_TEST_INT", tc.value)
			if got := envInt("FOGCAST_TEST_INT", 7); got != tc.want {
				t.Fatalf("envInt() = %d, want %d", got, tc.want)
			}
		})
	}
}
