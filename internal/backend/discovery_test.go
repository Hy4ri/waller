package backend

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetWallpapers verifies finding image files in a directory.
func TestGetWallpapers(t *testing.T) {
	// Arrange: Setup a dummy directory with specific files
	tmpDir := t.TempDir()

	// Create dummy files
	dummyImages := []string{"test1.jpg", "test2.png", "test3.jpeg"}
	for _, f := range dummyImages {
		fPath := filepath.Join(tmpDir, f)
		err := os.WriteFile(fPath, []byte("fake content"), 0644)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}

	// Create a non-image file that should be ignored
	nonImage := filepath.Join(tmpDir, "ignored.txt")
	os.WriteFile(nonImage, []byte("ignore me"), 0644)

	// Act
	images, err := GetWallpapers(tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(images) != 3 {
		t.Errorf("Expected 3 images, found %d", len(images))
	}
}
