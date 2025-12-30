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
	randomInterval := flag.Int("random", 0, "Interval in seconds to rotate wallpapers randomly")

	flag.Parse()

	// Daemon Mode (Wallpaper Window, CGO)
	if *daemonFlag != "" {
		layer.RunDaemon(*daemonFlag, *monitorIdxFlag)
		return
	}

	// Randomizer Mode
	if *randomInterval > 0 {
		manager.Init() // Init GTK for monitors

		cfg, err := config.Load()
		if err != nil {
			log.Fatal("Could not load config:", err)
		}
		if cfg.WallpaperDir == "" {
			log.Fatal("No wallpaper directory configured. Please run GUI first.")
		}

		fmt.Printf("Starting randomizer: dir=%s interval=%ds\n", cfg.WallpaperDir, *randomInterval)

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

			time.Sleep(time.Duration(*randomInterval) * time.Second)
		}
	}

	// GUI Mode (Manager, GTK)
	if err := gui.Run(); err != nil {
		fmt.Printf("Error running GUI: %v\n", err)
		os.Exit(1)
	}
}
