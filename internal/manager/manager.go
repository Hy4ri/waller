// Package manager handles wallpaper application logic including daemon process management.
// It uses IPC via Unix sockets to communicate with running daemons, only spawning new ones when needed.
package manager

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"
	"waller/internal/ipc"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// Init must be called to ensure GTK/GDK is initialized for monitor detection
func Init() {
	if err := gtk.InitCheck(nil); err != nil {
		// Fallback for non-GUI context if needed, but we need Display for monitors
		log.Println("Warning: GTK Init failed, monitor detection may fail")
	}
}

// ApplyWallpaper sets the wallpaper on the specified monitor index (-1 for All).
// It uses IPC to communicate with running daemons, only spawning new ones if needed.
func ApplyWallpaper(path string, monitorIndex int) {
	if monitorIndex == -1 {
		// Apply to ALL monitors
		display, _ := gdk.DisplayGetDefault()
		nMonitors := display.GetNMonitors()

		for i := 0; i < nMonitors; i++ {
			applyToMonitor(path, i)
		}
	} else {
		applyToMonitor(path, monitorIndex)
	}
}

// applyToMonitor handles wallpaper application for a single monitor.
// Uses IPC if daemon is running, spawns new daemon otherwise.
func applyToMonitor(path string, monitorIdx int) {
	socketPath := ipc.GetSocketPath(monitorIdx)

	// Check if daemon socket exists (daemon is running)
	if _, err := os.Stat(socketPath); err == nil {
		// Daemon is running, send update via IPC
		if sendIPCUpdate(socketPath, path) {
			return // Success
		}
		// IPC failed, socket might be stale - clean up and spawn new daemon
		os.Remove(socketPath)
	}

	// No running daemon, spawn a new one
	spawnDaemon(path, monitorIdx)
}

// sendIPCUpdate sends a wallpaper path to the daemon via Unix socket.
// Returns true on success, false on failure.
func sendIPCUpdate(socketPath, imagePath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		log.Printf("IPC dial failed: %v", err)
		return false
	}
	defer conn.Close()

	// Set write deadline to avoid hanging
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	// Send path with newline terminator (daemon expects this)
	_, err = fmt.Fprintf(conn, "%s\n", imagePath)
	if err != nil {
		log.Printf("IPC write failed: %v", err)
		return false
	}

	return true
}

// spawnDaemon starts a new daemon process for the specified monitor.
func spawnDaemon(path string, monitorIdx int) {
	self, _ := os.Executable()
	cmd := exec.Command(self, "--daemon", path, "--monitor-index", fmt.Sprintf("%d", monitorIdx))

	err := cmd.Start()
	if err != nil {
		log.Println("Failed to start daemon:", err)
		return
	}

	// Release the process so it continues independently
	cmd.Process.Release()

	// Wait briefly for daemon to create its socket
	socketPath := ipc.GetSocketPath(monitorIdx)
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		if _, err := os.Stat(socketPath); err == nil {
			return // Socket created, daemon is ready
		}
	}
}
