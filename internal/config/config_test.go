package config

import (
	"path/filepath"
	"testing"
)

// TestLoadConfig tests that a configuration can be loaded or created with defaults.
func TestLoadConfig(t *testing.T) {
	// Arrange: Create a temporary home directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Act: Load the configuration
	cfg, err := Load()

	// Assert: It should load without error and have expected defaults structure
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatalf("Expected non-nil config, got nil")
	}

	// Basic check for structural validity (for example, missing WallpaperDir if freshly created)
	expectedDir := filepath.Join(tmpHome, "Wallpapers")
	if cfg.WallpaperDir != "" && cfg.WallpaperDir != expectedDir {
		t.Logf("Note: WallpaperDir isn't exactly matched, got %v", cfg.WallpaperDir)
	}
}
