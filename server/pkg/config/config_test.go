package config

import (
	"log/slog"
	"testing"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("HTTP_ADDR", "")

	got, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v, want nil", err)
	}

	if got.AppEnv != "dev" {
		t.Errorf("AppEnv = %q, want %q", got.AppEnv, "dev")
	}
	if got.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", got.LogLevel, slog.LevelInfo)
	}
	if got.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", got.HTTPAddr, ":8080")
	}
}

func TestLoadFromEnv_AppEnv_Valid(t *testing.T) {
	tests := []struct {
		name   string
		appEnv string
		want   string
	}{
		{name: "dev", appEnv: "dev", want: "dev"},
		{name: "prod", appEnv: "prod", want: "prod"},
		{name: "dev with whitespace", appEnv: "  dev  ", want: "dev"},
		{name: "prod with whitespace", appEnv: "\nprod\t", want: "prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("LOG_LEVEL", "") // default
			t.Setenv("HTTP_ADDR", "") // default

			got, err := LoadFromEnv()
			if err != nil {
				t.Fatalf("LoadFromEnv() error = %v, want nil", err)
			}
			if got.AppEnv != tt.want {
				t.Errorf("AppEnv = %q, want %q", got.AppEnv, tt.want)
			}
		})
	}
}

func TestLoadFromEnv_AppEnv_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		appEnv string
	}{
		{name: "staging", appEnv: "staging"},
		{name: "empty-ish whitespace becomes default? no, trims to empty then default dev (so not invalid) - use real invalid", appEnv: "qa"},
		{name: "uppercase invalid", appEnv: "DEV"}, // note: code does not lower-case APP_ENV
		{name: "random", appEnv: "whatever"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("LOG_LEVEL", "")
			t.Setenv("HTTP_ADDR", "")

			_, err := LoadFromEnv()
			if err == nil {
				t.Fatalf("LoadFromEnv() error = nil, want non-nil")
			}
		})
	}
}

func TestLoadFromEnv_HTTPAddr(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "default when empty", in: "", want: ":8080"},
		{name: "trims whitespace", in: "  :9090  ", want: ":9090"},
		{name: "host:port", in: "127.0.0.1:8081", want: "127.0.0.1:8081"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", "")   // default dev
			t.Setenv("LOG_LEVEL", "") // default info
			t.Setenv("HTTP_ADDR", tt.in)

			got, err := LoadFromEnv()
			if err != nil {
				t.Fatalf("LoadFromEnv() error = %v, want nil", err)
			}
			if got.HTTPAddr != tt.want {
				t.Errorf("HTTPAddr = %q, want %q", got.HTTPAddr, tt.want)
			}
		})
	}
}

func TestParseLogLevel_Valid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want slog.Level
	}{
		{name: "debug", in: "debug", want: slog.LevelDebug},
		{name: "info", in: "info", want: slog.LevelInfo},
		{name: "warn", in: "warn", want: slog.LevelWarn},
		{name: "warning", in: "warning", want: slog.LevelWarn},
		{name: "error", in: "error", want: slog.LevelError},
		{name: "case insensitive", in: "DeBuG", want: slog.LevelDebug},
		{name: "trims whitespace", in: "  warn \n", want: slog.LevelWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLogLevel(tt.in)
			if err != nil {
				t.Fatalf("parseLogLevel(%q) error = %v, want nil", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseLogLevel_Invalid(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty string", in: ""},
		{name: "garbage", in: "nope"},
		{name: "almost warn", in: "warns"},
		{name: "numeric", in: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLogLevel(tt.in)
			if err == nil {
				t.Fatalf("parseLogLevel(%q) error = nil, want non-nil", tt.in)
			}
			// For invalid inputs, function returns LevelInfo along with an error.
			if got != slog.LevelInfo {
				t.Errorf("parseLogLevel(%q) = %v, want %v on error", tt.in, got, slog.LevelInfo)
			}
		})
	}
}

func TestLoadFromEnv_LogLevel_ValidAndInvalid(t *testing.T) {
	t.Run("valid LOG_LEVEL propagates", func(t *testing.T) {
		t.Setenv("APP_ENV", "dev")
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("HTTP_ADDR", "")

		got, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("LoadFromEnv() error = %v, want nil", err)
		}
		if got.LogLevel != slog.LevelDebug {
			t.Errorf("LogLevel = %v, want %v", got.LogLevel, slog.LevelDebug)
		}
	})

	t.Run("invalid LOG_LEVEL returns error", func(t *testing.T) {
		t.Setenv("APP_ENV", "dev")
		t.Setenv("LOG_LEVEL", "loud")
		t.Setenv("HTTP_ADDR", "")

		_, err := LoadFromEnv()
		if err == nil {
			t.Fatalf("LoadFromEnv() error = nil, want non-nil")
		}
	})
}
