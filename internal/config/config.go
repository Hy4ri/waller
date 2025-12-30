package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type BackendType string

const (
	BackendSwww      BackendType = "swww"
	BackendHyprpaper BackendType = "hyprpaper"
	BackendSwaybg    BackendType = "swaybg"
	BackendGsettings BackendType = "gsettings"
	BackendCustom    BackendType = "custom"
)

// Config holds the application settings that are saved to disk.
// The fields are tagged with `json:"..."` to define how they look in the config file.
type Config struct {
	// WallpaperDir is the path where the user stores their wallpapers.
	WallpaperDir string `json:"wallpaper_dir"`

	// PreferredBackend is the tool we use to set the wallpaper (e.g., swww, swaybg).
	PreferredBackend BackendType `json:"preferred_backend"`

	// CustomCommand allows the user to define their own command if they choose BackendCustom.
	// The placeholder %f will be replaced by the wallpaper path.
	CustomCommand string `json:"custom_command"`

	// Scaling determines how the image fits the screen (fill, fit, stretch).
	Scaling string `json:"scaling"`
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

	// Check if the file exists. If not, we return a default configuration
	// so the app can start fresh without crashing.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{
			WallpaperDir:     "",          // User will set this later
			PreferredBackend: BackendSwww, // Default to swww as it's the most feature-rich
			Scaling:          "fill",
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
