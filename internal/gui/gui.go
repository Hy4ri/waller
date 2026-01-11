// Package gui implements the GTK3-based wallpaper manager interface.
// It provides a visual grid of wallpapers with monitor selection and random rotation.
package gui

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"waller/internal/backend" // We can reuse the scanner logic
	"waller/internal/cache"
	"waller/internal/config"
	"waller/internal/manager"
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

	// Config
	cfg, _ := config.Load()

	// Main Layout
	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	win.Add(vbox)

	// Toolbar or Header
	header, _ := gtk.HeaderBarNew()
	header.SetShowCloseButton(false)
	header.SetTitle("Waller")
	win.SetTitlebar(header)

	// Directory Button
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

	// Dynamic Monitor Detection
	display, _ := gdk.DisplayGetDefault()
	nMonitors := display.GetNMonitors()

	monitorCombo.AppendText("All") // Maps to -1 for "All Monitors" logic
	for i := 0; i < nMonitors; i++ {
		// Try to get model/name if possible, else "Monitor N"
		mon, _ := display.GetMonitor(i)
		name := mon.GetModel()
		if name == "" {
			name = fmt.Sprintf("Monitor %d", i)
		}
		monitorCombo.AppendText(name) // Index i+1
	}
	monitorCombo.SetActive(0)

	header.PackStart(monitorCombo) // Add next to directory button

	// Global access for apply
	selectedMonitorIndex = -1 // -1 means All
	monitorCombo.Connect("changed", func() {
		active := monitorCombo.GetActive()
		if active == 0 {
			selectedMonitorIndex = -1
		} else {
			selectedMonitorIndex = active - 1 // 0-based index for GDK
		}
	})

	// Random Button
	randBtn, _ := gtk.ButtonNewWithLabel("Random")
	randBtn.Connect("clicked", func() {
		if len(globalFiles) > 0 {
			ri := rand.IntN(len(globalFiles))
			applyWallpaper(globalFiles[ri])
		}
	})
	header.PackEnd(randBtn)

	// Scroll Window for Grid
	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	vbox.PackStart(scroll, true, true, 0)

	// FlowBox for Grid
	flowBox, _ := gtk.FlowBoxNew()
	flowBox.SetVAlign(gtk.ALIGN_START)
	flowBox.SetMaxChildrenPerLine(30) // Dynamic really
	flowBox.SetSelectionMode(gtk.SELECTION_NONE)
	scroll.Add(flowBox)

	// We need global access to flowbox for loading
	// Store reference for async background loading
	globalFlowBox = flowBox

	win.ShowAll()

	if cfg.WallpaperDir != "" {
		loadWallpapers(cfg.WallpaperDir)
	}

	gtk.Main()
	return nil
}

var (
	globalFlowBox        *gtk.FlowBox
	globalFiles          []string
	selectedMonitorIndex int
)

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
		// Store globally for random
		globalFiles = files

		// Batch load to UI
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

	// Image
	// Use cache!
	thumbPath, err := cache.GetThumbnail(path, true)
	if err != nil {
		thumbPath = path // Fallback to full image
		// Trigger gen
		go cache.GetThumbnail(path, false)
	}

	pixbuf, _ := gdk.PixbufNewFromFileAtScale(thumbPath, 180, 130, true)
	img, _ := gtk.ImageNewFromPixbuf(pixbuf)
	img.Show() // Show widget explicitly instead of using deprecated ShowAll

	// Wrap image in a button to make it clickable
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

func applyWallpaper(path string) {
	manager.ApplyWallpaper(path, selectedMonitorIndex)
}
