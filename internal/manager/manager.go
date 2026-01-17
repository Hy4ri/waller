// Package manager handles wallpaper application logic including daemon process management.
// It uses IPC via a single Unix socket to communicate with the daemon.
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
		log.Println("Warning: GTK Init failed, monitor detection may fail")
	}
}

// ApplyWallpaper sets the wallpaper on the specified monitor index (-1 for All).
func ApplyWallpaper(path string, monitorIndex int) {
	// Ensure daemon is running
	ensureDaemonRunning(path)

	// Send update via IPC
	sendIPCUpdate(monitorIndex, path)
}

// ensureDaemonRunning checks if daemon is running, spawns if not.
func ensureDaemonRunning(initialPath string) {
	if _, err := os.Stat(ipc.SocketPath); err == nil {
		// Socket exists, try to connect
		conn, err := net.DialTimeout("unix", ipc.SocketPath, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return // Daemon is running
		}
		// Socket exists but can't connect - stale socket
		os.Remove(ipc.SocketPath)
	}

	// Spawn daemon
	spawnDaemon(initialPath)
}

// sendIPCUpdate sends a wallpaper path to the daemon via Unix socket.
func sendIPCUpdate(monitorIndex int, imagePath string) {
	conn, err := net.DialTimeout("unix", ipc.SocketPath, 2*time.Second)
	if err != nil {
		log.Printf("IPC dial failed: %v", err)
		return
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	// Send message in format "monitor:path"
	msg := ipc.FormatMessage(monitorIndex, imagePath)
	_, err = fmt.Fprintf(conn, "%s\n", msg)
	if err != nil {
		log.Printf("IPC write failed: %v", err)
	}
}

// spawnDaemon starts a new daemon process.
func spawnDaemon(path string) {
	self, _ := os.Executable()
	cmd := exec.Command(self, "--daemon", path)

	err := cmd.Start()
	if err != nil {
		log.Println("Failed to start daemon:", err)
		return
	}

	cmd.Process.Release()

	// Wait for daemon to create socket
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		if _, err := os.Stat(ipc.SocketPath); err == nil {
			return
		}
	}
}

// GetMonitorCount returns the number of connected monitors.
func GetMonitorCount() int {
	display, err := gdk.DisplayGetDefault()
	if err != nil {
		return 1
	}
	return display.GetNMonitors()
}
