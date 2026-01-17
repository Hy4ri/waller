// Package ipc provides shared utilities for inter-process communication
// between the manager and daemon processes via Unix sockets.
package ipc

import "fmt"

// GetSocketPath returns the Unix socket path for a given monitor index.
// This is the shared location used by both the manager and layer daemon
// to coordinate wallpaper updates.
func GetSocketPath(monitorIndex int) string {
	return fmt.Sprintf("/tmp/waller-%d.sock", monitorIndex)
}
