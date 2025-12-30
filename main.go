package main

import (
	"flag"
	"fmt"
	"os"

	"waller/internal/gui"
	"waller/internal/layer"
)

func main() {
	// Parse CLI flags
	daemonFlag := flag.String("daemon", "", "Start wallpaper daemon with image path")
	monitorIdxFlag := flag.Int("monitor-index", -1, "Monitor index to display on")

	flag.Parse()

	// Daemon Mode (Wallpaper Window, CGO)
	if *daemonFlag != "" {
		layer.RunDaemon(*daemonFlag, *monitorIdxFlag)
		return
	}

	// GUI Mode (Manager, GTK)
	if err := gui.Run(); err != nil {
		fmt.Printf("Error running GUI: %v\n", err)
		os.Exit(1)
	}
}
