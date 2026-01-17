// Package ipc provides shared utilities for inter-process communication
// between the manager and daemon processes via Unix sockets.
package ipc

import "fmt"

// SocketPath is the single Unix socket path for daemon communication.
const SocketPath = "/tmp/waller.sock"

// GetSocketPath returns the Unix socket path (kept for compatibility during transition).
// Deprecated: Use SocketPath constant directly.
func GetSocketPath(monitorIndex int) string {
	return SocketPath
}

// FormatMessage creates an IPC message in the format "monitor:path".
// Use monitor -1 for all monitors.
func FormatMessage(monitorIndex int, imagePath string) string {
	return fmt.Sprintf("%d:%s", monitorIndex, imagePath)
}

// ParseMessage parses an IPC message into monitor index and path.
// Returns monitor index and image path.
func ParseMessage(msg string) (int, string) {
	var monitor int
	var path string
	fmt.Sscanf(msg, "%d:%s", &monitor, &path)
	// Handle paths with colons by finding first colon
	for i, c := range msg {
		if c == ':' {
			fmt.Sscanf(msg[:i], "%d", &monitor)
			path = msg[i+1:]
			break
		}
	}
	return monitor, path
}
