package gui

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"waller/internal/backend" // We can reuse the scanner logic
	"waller/internal/cache"
	"waller/internal/config"
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
	header.SetShowCloseButton(true)
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

	monitorCombo.AppendText("All") // Index 0 in combo, but -1 for logic?
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
	// Closure hack?
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
	// Map monitor index -> PID of running daemon
	monitorDaemons = make(map[int]int)
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
				globalFlowBox.ShowAll()
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
		thumbPath = path // Fallback, trigger gen later?
		// Trigger gen
		go cache.GetThumbnail(path, false)
	}

	pixbuf, _ := gdk.PixbufNewFromFileAtScale(thumbPath, 180, 130, true)
	img, _ := gtk.ImageNewFromPixbuf(pixbuf)

	// Wrap image in a button to make it clickable
	imgBtn, _ := gtk.ButtonNew()
	imgBtn.SetRelief(gtk.RELIEF_NONE)
	imgBtn.Add(img)
	imgBtn.Connect("clicked", func() {
		applyWallpaper(path)
	})

	vbox.PackStart(imgBtn, true, true, 0)

	// Label
	name := filepath.Base(path)
	if len(name) > 15 {
		name = name[:12] + "..."
	}
	lbl, _ := gtk.LabelNew(name)
	vbox.PackStart(lbl, false, false, 0)

	globalFlowBox.Add(vbox)
}

func applyWallpaper(path string) {
	if selectedMonitorIndex == -1 {
		// Apply to ALL monitors: Kill all known daemons and respawn
		for idx, pid := range monitorDaemons {
			killDaemon(pid)
			delete(monitorDaemons, idx)
		}

		// Fallback cleanup just in case
		exec.Command("pkill", "-f", "waller --daemon").Run()

		display, _ := gdk.DisplayGetDefault()
		nMonitors := display.GetNMonitors()

		for i := 0; i < nMonitors; i++ {
			pid := spawnDaemon(path, i)
			if pid > 0 {
				monitorDaemons[i] = pid
			}
		}
	} else {
		// Specific monitor: Only kill the daemon for THIS monitor
		if pid, exists := monitorDaemons[selectedMonitorIndex]; exists {
			killDaemon(pid)
			delete(monitorDaemons, selectedMonitorIndex)
		}

		pid := spawnDaemon(path, selectedMonitorIndex)
		if pid > 0 {
			monitorDaemons[selectedMonitorIndex] = pid
		}
	}
}

func killDaemon(pid int) {
	p, err := os.FindProcess(pid)
	if err == nil {
		p.Kill()
	}
}

func spawnDaemon(path string, monitorIdx int) int {
	self, _ := os.Executable()
	cmd := exec.Command(self, "--daemon", path, "--monitor-index", fmt.Sprintf("%d", monitorIdx))
	err := cmd.Start()
	if err != nil {
		log.Println("Failed to start daemon:", err)
		return 0
	}

	// We do NOT release, so we can track it?
	// Actually, Releasing removes resources but PID remains until wait?
	// On Linux, if we don't Wait, it becomes a zombie until init reaps it.
	// But we want it to run indefinitely.
	// For this simple manager, just tracking PID is fine.
	cmd.Process.Release()
	return cmd.Process.Pid
}
