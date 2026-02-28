// Package ipc provides shared utilities for inter-process communication
// between the manager and daemon processes via Unix sockets.
package ipc

import (
	"strconv"
	"strings"
)

// SocketPath is the single Unix socket path for daemon communication.
const SocketPath = "/tmp/waller.sock"

// FormatMessage creates an IPC message in the format "monitor:path".
// Use monitor -1 for all monitors.
func FormatMessage(monitorIndex int, imagePath string) string {
	return strconv.Itoa(monitorIndex) + ":" + imagePath
}

// ParseMessage parses an IPC message ("monitor:path") into its components.
// Returns monitor index and image path.
func ParseMessage(msg string) (int, string) {
	before, after, ok := strings.Cut(msg, ":")
	if !ok {
		return -1, ""
	}

	monitor, err := strconv.Atoi(before)
	if err != nil {
		return -1, ""
	}

	return monitor, after
}
