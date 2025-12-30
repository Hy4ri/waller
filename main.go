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
		manager.Init() // Init GTK for monitors

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

		ri := rand.IntN(len(files))
		selected := files[ri]

		// Apply to specified monitor, or all monitors if not specified (-1)
		manager.ApplyWallpaper(selected, *monitorIdxFlag)

		fmt.Printf("Applied random wallpaper: %s\n", selected)
		return
	}

	// Auto-Rotation Mode
	if *autoInterval > 0 {
		manager.Init() // Init GTK for monitors

		cfg, err := config.Load()
		if err != nil {
			log.Fatal("Could not load config:", err)
		}
		if cfg.WallpaperDir == "" {
			log.Fatal("No wallpaper directory configured. Please run GUI first.")
		}

		fmt.Printf("Starting auto-rotation: dir=%s interval=%ds\n", cfg.WallpaperDir, *autoInterval)

		for {
			files, err := backend.GetWallpapers(cfg.WallpaperDir)
			if err != nil {
				log.Println("Error scanning wallpapers:", err)
			} else if len(files) > 0 {
				ri := rand.IntN(len(files))
				selected := files[ri]
				// Apply to ALL monitors (-1) by default for now
				manager.ApplyWallpaper(selected, -1)
			}

			time.Sleep(time.Duration(*autoInterval) * time.Second)
		}
	}

	// GUI Mode (Manager, GTK)
	if err := gui.Run(); err != nil {
		fmt.Printf("Error running GUI: %v\n", err)
		os.Exit(1)
	}
}
