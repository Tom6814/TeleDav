package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(map[string]string{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.DatabasePath != "data/app.db" {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, "data/app.db")
	}
	if cfg.StagingDir != "data/staging" {
		t.Fatalf("StagingDir = %q, want %q", cfg.StagingDir, "data/staging")
	}
	if cfg.WebDir != "web/build/web" {
		t.Fatalf("WebDir = %q, want %q", cfg.WebDir, "web/build/web")
	}
}
