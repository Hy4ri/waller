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

    // Load Image
    GtkWidget *image = gtk_image_new_from_file(image_path);

    // We need to scale the image to fill. GTK3 GtkImage doesn't scale easily without custom draw.
    // Let's use CSS or Pixbuf scaling.
    // Pixbuf approach:
    GError *err = NULL;
    GdkPixbuf *pixbuf = gdk_pixbuf_new_from_file(image_path, &err);
    if (err != NULL) {
        printf("Error loading image: %s\n", err->message);
        return window; // Start empty?
    }

    // Get screen size?
    // Ideally we scale on resize event.
    // For simplicity in this iteration: load as-is or use CSS cover.

    // CSS is cleaner for filling background.
    GtkStyleContext *context = gtk_widget_get_style_context(window);
    gtk_style_context_add_class(context, "wallpaper");

    // We'll return window and handle CSS in Go string
    // gtk_container_add(GTK_CONTAINER(window), image); // Placeholder, this won't scale

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

	// In a real app we'd iterate monitors and spawn a window for each.
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
