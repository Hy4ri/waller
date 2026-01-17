// Package layer provides the Wayland wallpaper daemon using gtk-layer-shell.
// It creates fullscreen windows on the background layer with CSS-scaled wallpapers.
// Single daemon handles all monitors via one IPC socket.
package layer

/*
#cgo pkg-config: gtk+-3.0 gtk-layer-shell-0

#include <gtk/gtk.h>
#include <gtk-layer-shell/gtk-layer-shell.h>

// Helper to create a layer window for a specific monitor
GtkWidget* create_wallpaper_window(int monitor_index) {
    GtkWidget *window = gtk_window_new(GTK_WINDOW_TOPLEVEL);

    gtk_layer_init_for_window(GTK_WINDOW(window));
    gtk_layer_set_layer(GTK_WINDOW(window), GTK_LAYER_SHELL_LAYER_BACKGROUND);

    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_LEFT, 1);
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_RIGHT, 1);
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_TOP, 1);
    gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_BOTTOM, 1);

    gtk_layer_set_exclusive_zone(GTK_WINDOW(window), -1);

    GdkDisplay *display = gdk_display_get_default();
    if (monitor_index >= 0 && monitor_index < gdk_display_get_n_monitors(display)) {
        GdkMonitor *monitor = gdk_display_get_monitor(display, monitor_index);
        gtk_layer_set_monitor(GTK_WINDOW(window), monitor);
    }

    GtkStyleContext *context = gtk_widget_get_style_context(window);
    gtk_style_context_add_class(context, "wallpaper");

    return window;
}

// Get number of monitors
int get_monitor_count() {
    GdkDisplay *display = gdk_display_get_default();
    return gdk_display_get_n_monitors(display);
}

// Per-window CSS providers to prevent memory leaks
typedef struct {
    GtkCssProvider *provider;
} WindowData;

static GHashTable *window_providers = NULL;

void init_window_providers() {
    if (window_providers == NULL) {
        window_providers = g_hash_table_new(g_direct_hash, g_direct_equal);
    }
}

void apply_css_to_window(GtkWidget* window, const char* css_data) {
    GtkStyleContext *context = gtk_widget_get_style_context(window);

    // Get or create provider for this window
    GtkCssProvider *provider = g_hash_table_lookup(window_providers, window);
    if (provider != NULL) {
        gtk_style_context_remove_provider(context, GTK_STYLE_PROVIDER(provider));
        g_object_unref(provider);
    }

    provider = gtk_css_provider_new();
    gtk_css_provider_load_from_data(provider, css_data, -1, NULL);
    gtk_style_context_add_provider(context, GTK_STYLE_PROVIDER(provider), GTK_STYLE_PROVIDER_PRIORITY_USER);
    g_hash_table_insert(window_providers, window, provider);
}

// Callback data for idle function
typedef struct {
    GtkWidget *window;
    char *css_data;
} IdleData;

static gboolean idle_update_wallpaper(gpointer user_data) {
    IdleData *data = (IdleData *)user_data;
    apply_css_to_window(data->window, data->css_data);
    g_free(data->css_data);
    g_free(data);
    return G_SOURCE_REMOVE;
}

void schedule_wallpaper_update(GtkWidget *window, const char *css_data) {
    IdleData *data = g_malloc(sizeof(IdleData));
    data->window = window;
    data->css_data = g_strdup(css_data);
    g_idle_add(idle_update_wallpaper, data);
}
*/
import "C"
import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"unsafe"

	"waller/internal/ipc"
)

// windows holds GTK window pointers for each monitor
var windows []*C.GtkWidget

// updateWallpaperCSS applies CSS to a window (for initial setup).
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

// scheduleWallpaperUpdate schedules a CSS update on the GTK main thread.
func scheduleWallpaperUpdate(win *C.GtkWidget, imagePath string) {
	css := fmt.Sprintf(`
        .wallpaper {
            background-image: url("%s");
            background-size: cover;
            background-repeat: no-repeat;
            background-position: center;
        }
    `, imagePath)

	cCss := C.CString(css)
	C.schedule_wallpaper_update(win, cCss)
	C.free(unsafe.Pointer(cCss))
}

// applyToMonitor applies wallpaper to specified monitor or all monitors.
func applyToMonitor(monitorIdx int, imagePath string) {
	if monitorIdx == -1 {
		// Apply to all monitors
		for _, win := range windows {
			scheduleWallpaperUpdate(win, imagePath)
		}
	} else if monitorIdx >= 0 && monitorIdx < len(windows) {
		scheduleWallpaperUpdate(windows[monitorIdx], imagePath)
	}

	// Force GC and release memory back to OS
	runtime.GC()
	debug.FreeOSMemory()
}

// RunDaemon starts the GTK main loop and displays wallpapers on all monitors.
// Single daemon handles all monitors via one IPC socket.
func RunDaemon(imagePath string, _ int) {
	debug.SetGCPercent(50)

	C.gtk_init(nil, nil)
	C.init_window_providers()

	// Create windows for all monitors
	nMonitors := int(C.get_monitor_count())
	windows = make([]*C.GtkWidget, nMonitors)

	for i := 0; i < nMonitors; i++ {
		win := C.create_wallpaper_window(C.int(i))
		windows[i] = win
		updateWallpaperCSS(win, imagePath)
		C.gtk_widget_show_all(win)
	}

	log.Printf("Daemon started: %d monitors, socket: %s", nMonitors, ipc.SocketPath)

	// Setup single IPC socket
	os.Remove(ipc.SocketPath)
	listener, err := net.Listen("unix", ipc.SocketPath)
	if err != nil {
		log.Printf("Warning: Failed to create IPC socket: %v", err)
	} else {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			listener.Close()
			os.Remove(ipc.SocketPath)
			os.Exit(0)
		}()

		go func() {
			defer listener.Close()
			defer os.Remove(ipc.SocketPath)

			buf := make([]byte, 1024)

			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}

				n, err := conn.Read(buf)
				conn.Close()

				if err != nil || n == 0 {
					continue
				}

				msg := strings.TrimSpace(string(buf[:n]))
				if msg == "" {
					continue
				}

				// Parse message: "monitor:path"
				monitorIdx, path := ipc.ParseMessage(msg)
				if path != "" {
					applyToMonitor(monitorIdx, path)
				}
			}
		}()
	}

	C.gtk_main()
}
