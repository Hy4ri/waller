// Package cache provides thumbnail generation and caching for wallpaper images.
// Thumbnails are stored in the user's cache directory and indexed by MD5 hash.
package cache

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sync"

	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

// thumbDir is resolved once at first use to avoid repeated syscalls.
var (
	thumbDir     string
	thumbDirOnce sync.Once
	thumbDirErr  error
)

// initThumbDir resolves and creates the thumbnail cache directory.
func initThumbDir() {
	thumbDirOnce.Do(func() {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			thumbDirErr = err
			return
		}
		thumbDir = filepath.Join(cacheDir, "waller", "thumbnails")
		thumbDirErr = os.MkdirAll(thumbDir, 0755)
	})
}

// GetThumbnail returns the path to a cached thumbnail for the given image path.
// If the thumbnail does not exist, it generates one.
// fastCheck: if true, only checks existence, does not generate (returns error if missing).
func GetThumbnail(originalPath string, fastCheck bool) (string, error) {
	initThumbDir()
	if thumbDirErr != nil {
		return "", thumbDirErr
	}

	hash := md5.Sum([]byte(originalPath))
	hashStr := hex.EncodeToString(hash[:])
	thumbPath := filepath.Join(thumbDir, hashStr+".jpg")

	// Check if exists
	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	if fastCheck {
		return "", errors.New("thumbnail not found")
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

	// Resize to width 200 (preserving aspect ratio)
	// NearestNeighbor is faster and uses less memory than Lanczos3
	m := resize.Resize(200, 0, img, resize.NearestNeighbor)

	out, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Save as JPEG with quality 75 (reduces file size, imperceptible at thumbnail size)
	return thumbPath, jpeg.Encode(out, m, &jpeg.Options{Quality: 75})
}
