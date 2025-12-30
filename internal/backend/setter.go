package backend

import (
	"fmt"
	"os/exec"
	"strings"

	"waller/internal/config"
)

type WallpaperSetter interface {
	SetWallpaper(path string, monitor string) error
}

type Setter struct {
	cfg *config.Config
}

// NewSetter creates a new Setter instance with the given configuration.
func NewSetter(cfg *config.Config) *Setter {
	return &Setter{cfg: cfg}
}

// SetWallpaper applies the wallpaper at the given path to the specified monitor.
// If monitor is empty or "All", it attempts to apply it to all monitors.
func (s *Setter) SetWallpaper(path string, monitor string) error {
	// We switch on the PreferredBackend from the config to decide which tool to call.
	switch s.cfg.PreferredBackend {
	case config.BackendSwww:
		return s.setSwww(path, monitor)
	case config.BackendHyprpaper:
		return s.setHyprpaper(path, monitor)
	case config.BackendSwaybg:
		return s.setSwaybg(path, monitor)
	case config.BackendGsettings:
		return s.setGsettings(path) // GNOME usually sets global or primary
	case config.BackendCustom:
		return s.setCustom(path)
	default:
		return fmt.Errorf("unknown backend: %s", s.cfg.PreferredBackend)
	}
}

// setSwww uses the 'swww' command line tool.
// Docs: https://github.com/LGFae/swww
func (s *Setter) setSwww(path string, monitor string) error {
	// We use 'img' to set the image.
	// --transition-type grow: Makes it look nice.
	// --transition-pos 0.5,0.5: Grows from the center.
	args := []string{"img", path, "--transition-type", "grow", "--transition-pos", "0.5,0.5", "--transition-step", "90"}

	// If a specific monitor is targeted, add the flag.
	if monitor != "" && monitor != "All" {
		args = append(args, "--outputs", monitor)
	}

	// Execute the command
	cmd := exec.Command("swww", args...)
	return cmd.Run()
}

func (s *Setter) setSwaybg(path string, monitor string) error {
	// swaybg usually runs as a daemon. Restarting it or just running a new one?
	// Usually users run `swaybg -i ...` in config.
	// To change it dynamically, we might need to kill old swaybg or simple exec a new one.
	// A common trick is `pkill swaybg; swaybg -i ... &`
	// exec.Command won't work well if it needs to stay alive.
	// For "Manager" app, we might just spawn it and let it detach?

	// WARNING: primitive "killall" approach
	exec.Command("pkill", "swaybg").Run()

	mode := "fill"
	if s.cfg.Scaling != "" {
		mode = s.cfg.Scaling
	}

	args := []string{"-i", path, "-m", mode}
	if monitor != "" && monitor != "All" {
		args = append(args, "-o", monitor)
	}

	cmd := exec.Command("swaybg", args...)
	return cmd.Start() // Start async
}

func (s *Setter) setHyprpaper(path string, monitor string) error {
	// hyprpaper requires IPC.
	// 1. Preload: hyprctl hyprpaper preload <path>
	// 2. Wallpaper: hyprctl hyprpaper wallpaper <monitor>,<path>

	// Encode path just in case? Hyprctl handles spaces usually
	preloadCmd := exec.Command("hyprctl", "hyprpaper", "preload", path)
	if err := preloadCmd.Run(); err != nil {
		// Ignore check, might be already preloaded
	}

	target := monitor
	if target == "" || target == "All" {
		// Hyprpaper needs specific monitors usually, or special logic.
		// If "All", we might need to list monitors and loop.
		// For now, let's assume ",path" sets it for currently focused or default?
		// Actually hyprpaper syntax is "monitor,path"
		// If we don't know monitor, we can guess or error.
		// Let's default to comma for "all"? No, that's not standard.
		// Hack: use empty for all active?
		target = ","
	} else {
		target = monitor + ","
	}

	wallCmd := exec.Command("hyprctl", "hyprpaper", "wallpaper", target+path)
	return wallCmd.Run()
}

func (s *Setter) setGsettings(path string) error {
	// gsettings set org.gnome.desktop.background picture-uri-dark file://...
	uri := "file://" + path
	exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
	return exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri).Run()
}

func (s *Setter) setCustom(path string) error {
	if s.cfg.CustomCommand == "" {
		return fmt.Errorf("custom command is empty")
	}
	// Simple replacement of %f with file path
	cmdStr := strings.ReplaceAll(s.cfg.CustomCommand, "%f", path)
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("invalid custom command")
	}
	return exec.Command(parts[0], parts[1:]...).Run()
}
