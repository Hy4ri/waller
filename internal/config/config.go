// Package config manages application configuration including wallpaper directory settings.
// Configuration is stored as JSON in the user's config directory.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application settings that are saved to disk.
type Config struct {
	// WallpaperDir is the path where the user stores their wallpapers.
	WallpaperDir string `json:"wallpaper_dir"`
}

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// Use 'waller' as the configuration directory name
	return filepath.Join(configDir, "waller", "config.json"), nil
}

// Load reads the config file from disk.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if the file exists. If not, return a default configuration.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{
			WallpaperDir: "", // User will set this in the GUI
		}, nil
	}

	// Read the raw bytes from the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal (parse) the JSON bytes into our Config struct
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
