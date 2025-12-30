// Package manager handles wallpaper application logic including daemon process management.
// It spawns gtk-layer-shell daemon processes per monitor and tracks their PIDs.
package manager

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// We need to keep track of PIDs if we are a long-running process (like GUI or Randomizer)
// Map monitor index -> PID
var monitorDaemons = make(map[int]int)

// Init must be called to ensure GTK/GDK is initialized for monitor detection
func Init() {
	if err := gtk.InitCheck(nil); err != nil {
		// Fallback for non-GUI context if needed, but we need Display for monitors
		log.Println("Warning: GTK Init failed, monitor detection may fail")
	}
}

// ApplyWallpaper sets the wallpaper on the specified monitor index (-1 for All).
// It manages the daemon processes.
func ApplyWallpaper(path string, monitorIndex int) {
	if monitorIndex == -1 {
		// Apply to ALL monitors

		// 1. Cleanup existing tracked daemons
		for idx, pid := range monitorDaemons {
			killDaemon(pid)
			delete(monitorDaemons, idx)
		}
		// 2. Fallback cleanup (in case of external kills or restarts)
		exec.Command("pkill", "-f", "waller --daemon").Run()

		// 3. Detect and Spawn
		display, _ := gdk.DisplayGetDefault()
		nMonitors := display.GetNMonitors()

		for i := 0; i < nMonitors; i++ {
			pid := spawnDaemon(path, i)
			if pid > 0 {
				monitorDaemons[i] = pid
			}
		}
	} else {
		// Specific monitor
		if pid, exists := monitorDaemons[monitorIndex]; exists {
			killDaemon(pid)
			delete(monitorDaemons, monitorIndex)
		}

		pid := spawnDaemon(path, monitorIndex)
		if pid > 0 {
			monitorDaemons[monitorIndex] = pid
		}
	}
}

func killDaemon(pid int) {
	p, err := os.FindProcess(pid)
	if err == nil {
		p.Kill()
		p.Wait() // Avoid zombies
	}
}

func spawnDaemon(path string, monitorIdx int) int {
	self, _ := os.Executable()
	cmd := exec.Command(self, "--daemon", path, "--monitor-index", fmt.Sprintf("%d", monitorIdx))
	// We want the daemon to persist, so we start it and let it go.
	// However, we need its PID.
	err := cmd.Start()
	if err != nil {
		log.Println("Failed to start daemon:", err)
		return 0
	}

	// Detach? Use Release to let it live if we die?
	// If we are the Randomizer, we want them to die when WE die?
	// User expects "waller --random" to effectively "own" the desktop.
	// If we Release, they become orphans re-parented to init.
	// That is fine for a permanent wallpaper.
	cmd.Process.Release()
	return cmd.Process.Pid
}
