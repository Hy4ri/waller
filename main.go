package main

import (
	"flag"
	"fmt"
	"os"

	"log"
	"math/rand/v2"
	"time"

	"waller/internal/backend"
	"waller/internal/config"
	"waller/internal/gui"
	"waller/internal/layer"
	"waller/internal/manager"

	"github.com/gotk3/gotk3/gtk"
)

func main() {
	// Parse CLI flags
	daemonFlag := flag.String("daemon", "", "Start wallpaper daemon with image path")
	monitorIdxFlag := flag.Int("monitor-index", -1, "Monitor index to display on")
	autoInterval := flag.Int("auto", 0, "Interval in seconds to rotate wallpapers automatically")
	randomFlag := flag.Bool("random", false, "Apply a random wallpaper once")

	flag.Parse()

	// Daemon Mode (Wallpaper Window, CGO)
	if *daemonFlag != "" {
		layer.RunDaemon(*daemonFlag, *monitorIdxFlag)
		return
	}

	// Random Wallpaper Mode (one-time)
	if *randomFlag {
		files, _ := loadConfigAndGetWallpapers()
		ri := rand.IntN(len(files))
		selected := files[ri]

		manager.ApplyWallpaper(selected, *monitorIdxFlag)
		fmt.Printf("Applied random wallpaper: %s\n", selected)
		return
	}

	if *autoInterval > 0 {
		files, wallpaperDir := loadConfigAndGetWallpapers()
		fmt.Printf("Starting auto-rotation: dir=%s interval=%ds wallpapers=%d\n", wallpaperDir, *autoInterval, len(files))

		for {
			ri := rand.IntN(len(files))
			selected := files[ri]
			manager.ApplyWallpaper(selected, -1)
			time.Sleep(time.Duration(*autoInterval) * time.Second)
		}
	}

	if err := gui.Run(); err != nil {
		fmt.Printf("Error running GUI: %v\n", err)
		os.Exit(1)
	}
}

func loadConfigAndGetWallpapers() ([]string, string) {
	if err := gtk.InitCheck(nil); err != nil {
		log.Fatal("GTK init failed:", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Could not load config:", err)
	}
	if cfg.WallpaperDir == "" {
		log.Fatal("No wallpaper directory configured. Please run GUI first.")
	}

	files, err := backend.GetWallpapers(cfg.WallpaperDir)
	if err != nil {
		log.Fatal("Error scanning wallpapers:", err)
	}
	if len(files) == 0 {
		log.Fatal("No wallpapers found in directory")
	}

	return files, cfg.WallpaperDir
}
