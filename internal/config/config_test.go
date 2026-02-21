package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.General.Interval != 10 {
		t.Errorf("default interval = %d, want 10", cfg.General.Interval)
	}
	if cfg.General.Language != "en" {
		t.Errorf("default language = %q, want en", cfg.General.Language)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultConfig()
	cfg.General.Timezone = "Asia/Seoul"

	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.General.Timezone != "Asia/Seoul" {
		t.Errorf("timezone = %q, want Asia/Seoul", loaded.General.Timezone)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	os.WriteFile(path, []byte("{{invalid toml}}"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perms.toml")

	if err := Save(DefaultConfig(), path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestDefaultPath_NotEmpty(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Error("DefaultPath should not be empty")
	}
}
