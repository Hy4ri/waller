// Package gui implements the GTK3-based wallpaper manager interface.
// It provides a visual grid of wallpapers with monitor selection and random rotation.
package gui

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"waller/internal/backend"
	"waller/internal/cache"
	"waller/internal/config"
	"waller/internal/manager"
)

// Global state for async wallpaper loading and monitor selection
var (
	globalFlowBox        *gtk.FlowBox
	globalFiles          []string
	globalFilesMu        sync.Mutex
	selectedMonitorIndex int
)

func Run() error {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return err
	}
	win.SetTitle("Waller Manager")
	win.SetDefaultSize(800, 600)
	if _, err := os.Stat("icon.png"); err == nil {
		win.SetIconFromFile("icon.png")
	}
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: failed to load config: %v", err)
		cfg = &config.Config{}
	}

	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	win.Add(vbox)

	header, _ := gtk.HeaderBarNew()
	header.SetShowCloseButton(false)
	header.SetTitle("Waller")
	win.SetTitlebar(header)

	dirBtn, _ := gtk.ButtonNewWithLabel("Open Directory")
	dirBtn.Connect("clicked", func() {
		dlg, _ := gtk.FileChooserNativeDialogNew("Select Wallpaper Dir", win, gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER, "Select", "Cancel")
		resp := dlg.Run()
		if resp == int(gtk.RESPONSE_ACCEPT) {
			folder := dlg.GetFilename()
			cfg.WallpaperDir = folder
			cfg.Save()
			loadWallpapers(folder)
		}
		dlg.Destroy()
	})
	header.PackStart(dirBtn)
	// Monitor Selection
	monitorCombo, _ := gtk.ComboBoxTextNew()
	refreshMonitors(monitorCombo)

	header.PackStart(monitorCombo)

	selectedMonitorIndex = -1
	monitorCombo.Connect("changed", func() {
		active := monitorCombo.GetActive()
		if active == 0 {
			selectedMonitorIndex = -1
		} else {
			selectedMonitorIndex = active - 1 // 0-based index for GDK
		}
	})

	// Refresh Button — re-detects monitors and reloads wallpapers
	refreshBtn, _ := gtk.ButtonNewWithLabel("Refresh")
	refreshBtn.Connect("clicked", func() {
		refreshMonitors(monitorCombo)
		selectedMonitorIndex = -1

		if cfg.WallpaperDir != "" {
			loadWallpapers(cfg.WallpaperDir)
		}
	})
	header.PackStart(refreshBtn)

	randBtn, _ := gtk.ButtonNewWithLabel("Random")
	randBtn.Connect("clicked", func() {
		globalFilesMu.Lock()
		files := globalFiles
		globalFilesMu.Unlock()

		if len(files) > 0 {
			ri := rand.IntN(len(files))
			applyWallpaper(files[ri])
		}
	})
	header.PackEnd(randBtn)

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	vbox.PackStart(scroll, true, true, 0)

	flowBox, _ := gtk.FlowBoxNew()
	flowBox.SetVAlign(gtk.ALIGN_START)
	flowBox.SetMaxChildrenPerLine(30) // Dynamic really
	flowBox.SetSelectionMode(gtk.SELECTION_NONE)
	scroll.Add(flowBox)

	globalFlowBox = flowBox

	win.ShowAll()

	if cfg.WallpaperDir != "" {
		loadWallpapers(cfg.WallpaperDir)
	}

	gtk.Main()
	return nil
}

func loadWallpapers(dir string) {
	// Clear existing
	children := globalFlowBox.GetChildren()
	children.Foreach(func(item interface{}) {
		globalFlowBox.Remove(item.(*gtk.Widget))
	})

	go func() {
		files, err := backend.GetWallpapers(dir)
		if err != nil {
			log.Println("Error:", err)
			return
		}

		globalFilesMu.Lock()
		globalFiles = files
		globalFilesMu.Unlock()

		// Pre-generate missing thumbnails with a bounded worker pool
		numWorkers := runtime.NumCPU()
		jobs := make(chan string, numWorkers)

		var wg sync.WaitGroup
		for range numWorkers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for path := range jobs {
					cache.GetThumbnail(path, false)
				}
			}()
		}

		// Enqueue files that need thumbnails
		for _, path := range files {
			if _, err := cache.GetThumbnail(path, true); err != nil {
				jobs <- path
			}
		}
		close(jobs)
		wg.Wait()

		// Batch load to UI (thumbnails are all cached now)
		batchSize := 20
		total := len(files)

		for i := 0; i < total; i += batchSize {
			end := i + batchSize
			if end > total {
				end = total
			}

			batch := files[i:end]

			glib.IdleAdd(func() bool {
				for _, path := range batch {
					addWallpaperItem(path)
				}
				return false // Run once
			})
		}
	}()
}

func addWallpaperItem(path string) {
	// Container
	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)

	thumbPath, err := cache.GetThumbnail(path, true)
	if err != nil {
		thumbPath = path // Fallback to full image
	}

	pixbuf, _ := gdk.PixbufNewFromFileAtScale(thumbPath, 150, 100, true)
	img, _ := gtk.ImageNewFromPixbuf(pixbuf)
	// Free the pixbuf after creating the image to release memory
	if pixbuf != nil {
		pixbuf.Unref()
	}
	img.Show()

	imgBtn, _ := gtk.ButtonNew()
	imgBtn.SetRelief(gtk.RELIEF_NONE)
	imgBtn.Add(img)
	imgBtn.Connect("clicked", func() {
		applyWallpaper(path)
	})
	imgBtn.Show() // Show widget explicitly

	vbox.PackStart(imgBtn, true, true, 0)

	// Label
	name := filepath.Base(path)
	if len(name) > 15 {
		name = name[:12] + "..."
	}
	lbl, _ := gtk.LabelNew(name)
	lbl.Show() // Show widget explicitly
	vbox.PackStart(lbl, false, false, 0)

	vbox.Show() // Show container explicitly
	globalFlowBox.Add(vbox)
}

// refreshMonitors clears and repopulates the monitor combo box
// from the current GDK display state.
func refreshMonitors(combo *gtk.ComboBoxText) {
	combo.RemoveAll()
	combo.AppendText("All") // Index 0 → monitorIndex -1

	display, _ := gdk.DisplayGetDefault()
	nMonitors := display.GetNMonitors()

	for i := 0; i < nMonitors; i++ {
		mon, _ := display.GetMonitor(i)
		name := mon.GetModel()
		if name == "" {
			name = fmt.Sprintf("Monitor %d", i)
		}
		combo.AppendText(name)
	}
	combo.SetActive(0)
}

func applyWallpaper(path string) {
	manager.ApplyWallpaper(path, selectedMonitorIndex)
}
