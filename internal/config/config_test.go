package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Create a minimal YAML config file with only required fields.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
app:
  mode: worker
  name: testplane
database:
  dsn: "postgres://localhost/test"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Check explicit values.
	if cfg.App.Mode != "worker" {
		t.Errorf("App.Mode = %q, want %q", cfg.App.Mode, "worker")
	}
	if cfg.App.Name != "testplane" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "testplane")
	}
	if cfg.Database.DSN != "postgres://localhost/test" {
		t.Errorf("Database.DSN = %q, want %q", cfg.Database.DSN, "postgres://localhost/test")
	}

	// Check defaults.
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Database.MaxOpenConns = %d, want 25", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 10 {
		t.Errorf("Database.MaxIdleConns = %d, want 10", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 300 {
		t.Errorf("Database.ConnMaxLifetime = %d, want 300", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "localhost:6379")
	}
	if cfg.Redis.DB != 0 {
		t.Errorf("Redis.DB = %d, want 0", cfg.Redis.DB)
	}
	if cfg.TDLib.DataDir != "./data/sessions" {
		t.Errorf("TDLib.DataDir = %q, want %q", cfg.TDLib.DataDir, "./data/sessions")
	}
	if cfg.TDLib.LogLevel != 1 {
		t.Errorf("TDLib.LogLevel = %d, want 1", cfg.TDLib.LogLevel)
	}
	if cfg.GRPC.ListenAddr != ":50051" {
		t.Errorf("GRPC.ListenAddr = %q, want %q", cfg.GRPC.ListenAddr, ":50051")
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":8080")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.JSON != false {
		t.Errorf("Log.JSON = %v, want false", cfg.Log.JSON)
	}
	if cfg.RateLimit.RPS != 100 {
		t.Errorf("RateLimit.RPS = %f, want 100", cfg.RateLimit.RPS)
	}
	if cfg.RateLimit.Burst != 200 {
		t.Errorf("RateLimit.Burst = %d, want 200", cfg.RateLimit.Burst)
	}
}

func TestLoad_OverrideDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
app:
  mode: main
  name: custom
database:
  dsn: "postgres://localhost/custom"
  max_open_conns: 50
  max_idle_conns: 20
  conn_max_lifetime_seconds: 600
redis:
  addr: "redis.example.com:6379"
  password: "secret"
  db: 2
http:
  addr: ":9090"
log:
  level: debug
  json: true
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("Database.MaxOpenConns = %d, want 50", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 20 {
		t.Errorf("Database.MaxIdleConns = %d, want 20", cfg.Database.MaxIdleConns)
	}
	if cfg.Redis.Addr != "redis.example.com:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "redis.example.com:6379")
	}
	if cfg.Redis.Password != "secret" {
		t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, "secret")
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}
	if cfg.HTTP.Addr != ":9090" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":9090")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Log.JSON != true {
		t.Errorf("Log.JSON = %v, want true", cfg.Log.JSON)
	}
}

func TestLoad_InvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	// Use YAML that is structurally invalid (tab indentation mixed with mapping).
	badYAML := "key:\n\t- broken:\n  mixed: {[invalid"
	if err := os.WriteFile(cfgPath, []byte(badYAML), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
app:
  mode: main
database:
  dsn: "postgres://localhost/test"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Viper's AutomaticEnv + SetEnvKeyReplacer(".", "_") means env var
	// LOG_LEVEL maps to viper key "log.level". This works because "log.level"
	// already has a default set via SetDefault.
	t.Setenv("LOG_LEVEL", "error")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Log.Level != "error" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "error")
	}
}

func TestLoad_EnvOverrideHTTPAddr(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
app:
  mode: main
database:
  dsn: "postgres://localhost/test"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	t.Setenv("HTTP_ADDR", ":3000")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.HTTP.Addr != ":3000" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":3000")
	}
}
