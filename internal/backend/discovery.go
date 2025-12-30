// Package backend handles wallpaper file discovery and scanning.
// It finds all supported image files in a given directory.
package backend

import (
	"os"
	"path/filepath"
	"strings"
)

// validExtensions is a set (map for O(1) lookup) of supported image file types.
var validExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// GetWallpapers scans the given directory and returns a list of absolute paths
// for all supported image files found.
func GetWallpapers(dir string) ([]string, error) {
	var wallpapers []string

	// ReadDir reads the named directory and returns all its directory entries sorted by filename.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if validExtensions[ext] {
			wallpapers = append(wallpapers, filepath.Join(dir, entry.Name()))
		}
	}

	return wallpapers, nil
}
