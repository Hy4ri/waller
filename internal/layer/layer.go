// Package layer provides the Wayland wallpaper daemon using gtk-layer-shell.
// It creates fullscreen windows on the background layer with CSS-scaled wallpapers.
package layer

/*
#cgo pkg-config: gtk+-3.0 gtk-layer-shell-0

#include <gtk/gtk.h>
#include <gtk-layer-shell/gtk-layer-shell.h>

// Helper to create a layer window
GtkWidget* create_wallpaper_window(char* image_path, int monitor_index) {
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

void apply_css(GtkWidget* window, char* css_data) {
    GtkCssProvider *provider = gtk_css_provider_new();
    gtk_css_provider_load_from_data(provider, css_data, -1, NULL);

    GtkStyleContext *context = gtk_widget_get_style_context(window);
    gtk_style_context_add_provider(context, GTK_STYLE_PROVIDER(provider), GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
    gtk_style_context_add_class(context, "wallpaper");
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// RunDaemon starts the GTK main loop and displays the wallpaper.
func RunDaemon(imagePath string, monitorIndex int) {
	C.gtk_init(nil, nil)

	cPath := C.CString(imagePath)
	defer C.free(unsafe.Pointer(cPath))

	// Create wallpaper window for the specified monitor
	win := C.create_wallpaper_window(cPath, C.int(monitorIndex))

	// CSS for background scaling
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

	C.apply_css(win, cCss)

	C.gtk_widget_show_all(win)
	C.gtk_main()
}
