// Package layer provides the Wayland wallpaper daemon using gtk-layer-shell.
// It creates fullscreen windows on the background layer with CSS-scaled wallpapers.
// Supports IPC via Unix socket to update wallpapers without restarting.
package layer

/*
#cgo pkg-config: gtk+-3.0 gtk-layer-shell-0

#include <gtk/gtk.h>
#include <gtk-layer-shell/gtk-layer-shell.h>

// Helper to create a layer window
GtkWidget* create_wallpaper_window(int monitor_index) {
    GtkWidget *window = gtk_window_new(GTK_WINDOW_TOPLEVEL);

    gtk_layer_init_for_window(GTK_WINDOW(window));
    gtk_layer_set_layer(GTK_WINDOW(window), GTK_LAYER_SHELL_LAYER_BACKGROUND);

    // Anchor to all edges
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_LEFT, 1);
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_RIGHT, 1);
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_TOP, 1);
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_BOTTOM, 1);

    // Exclusive zone -1 to ignore
    gtk_layer_set_exclusive_zone(GTK_WINDOW(window), -1);

    // Set Monitor
    GdkDisplay *display = gdk_display_get_default();
    if (monitor_index >= 0 && monitor_index < gdk_display_get_n_monitors(display)) {
        GdkMonitor *monitor = gdk_display_get_monitor(display, monitor_index);
        gtk_layer_set_monitor(GTK_WINDOW(window), monitor);
    }

    // CSS is cleaner for filling background.
    GtkStyleContext *context = gtk_widget_get_style_context(window);
    gtk_style_context_add_class(context, "wallpaper");

    return window;
}

// Global CSS provider to track and release old wallpaper resources
static GtkCssProvider *current_provider = NULL;

// apply_css_to_window applies CSS styling with the given image path to the window.
// Properly cleans up the previous provider to prevent memory leaks from accumulated images.
void apply_css_to_window(GtkWidget* window, const char* css_data) {
    GtkStyleContext *context = gtk_widget_get_style_context(window);

    // Remove and unref the old provider to release previous image memory
    if (current_provider != NULL) {
        gtk_style_context_remove_provider(context, GTK_STYLE_PROVIDER(current_provider));
        g_object_unref(current_provider);
        current_provider = NULL;
    }

    // Create and apply new provider
    current_provider = gtk_css_provider_new();
    gtk_css_provider_load_from_data(current_provider, css_data, -1, NULL);
    gtk_style_context_add_provider(context, GTK_STYLE_PROVIDER(current_provider), GTK_STYLE_PROVIDER_PRIORITY_USER);
}
*/
import "C"
import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unsafe"
	"waller/internal/ipc"

	"github.com/gotk3/gotk3/glib"
)

// updateWallpaperCSS generates and applies new CSS for the given image path.
func updateWallpaperCSS(win *C.GtkWidget, imagePath string) {
	css := fmt.Sprintf(`
        .wallpaper {
            background-image: url("%s");
            background-size: cover;
            background-repeat: no-repeat;
            background-position: center;
        }
    `, imagePath)

	cCss := C.CString(css)
	defer C.free(unsafe.Pointer(cCss))
	C.apply_css_to_window(win, cCss)
}

// RunDaemon starts the GTK main loop and displays the wallpaper.
// It also starts a Unix socket IPC listener to accept wallpaper update commands.
func RunDaemon(imagePath string, monitorIndex int) {
	C.gtk_init(nil, nil)

	// Create wallpaper window for the specified monitor
	win := C.create_wallpaper_window(C.int(monitorIndex))

	// Apply initial wallpaper CSS
	updateWallpaperCSS(win, imagePath)

	C.gtk_widget_show_all(win)

	// Setup IPC socket
	socketPath := ipc.GetSocketPath(monitorIndex)
	os.Remove(socketPath) // Remove stale socket if exists

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Printf("Warning: Failed to create IPC socket: %v", err)
	} else {
		// Cleanup socket on shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			listener.Close()
			os.Remove(socketPath)
			os.Exit(0)
		}()

		// IPC listener goroutine
		go func() {
			defer listener.Close()
			defer os.Remove(socketPath)

			for {
				conn, err := listener.Accept()
				if err != nil {
					// Listener closed, exit goroutine
					return
				}

				// Read the new image path from the socket
				reader := bufio.NewReader(conn)
				newPath, err := reader.ReadString('\n')
				conn.Close()

				if err != nil {
					continue
				}

				newPath = strings.TrimSpace(newPath)
				if newPath == "" {
					continue
				}

				// Use glib.IdleAdd to safely update UI from this goroutine
				// Must capture newPath value to avoid race condition
				pathCopy := newPath
				glib.IdleAdd(func() bool {
					updateWallpaperCSS(win, pathCopy)
					return false // Run once
				})
			}
		}()
	}

	C.gtk_main()
}
