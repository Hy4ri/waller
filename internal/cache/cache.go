// Package cache provides thumbnail generation and caching for wallpaper images.
// Thumbnails are stored in the user's cache directory and indexed by MD5 hash.
package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

// GetThumbnail returns the path to a cached thumbnail for the given image path.
// If the thumbnail does not exist, it generates one.
// fastCheck: if true, only checks existence, does not generate (returns error if missing).
func GetThumbnail(originalPath string, fastCheck bool) (string, error) {
	hash := md5.Sum([]byte(originalPath))
	hashStr := hex.EncodeToString(hash[:])

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	thumbDir := filepath.Join(cacheDir, "waller", "thumbnails")
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", err
	}

	thumbPath := filepath.Join(thumbDir, hashStr+".jpg")

	// Check if exists
	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	if fastCheck {
		return "", fmt.Errorf("thumbnail not found")
	}

	// Generate
	file, err := os.Open(originalPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	// Resize to width 300 (preserving aspect ratio)
	// uint(300) width, 0 height means preserve aspect ratio
	m := resize.Resize(300, 0, img, resize.Lanczos3)

	out, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Save as JPEG
	return thumbPath, jpeg.Encode(out, m, nil)
}
