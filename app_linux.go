//go:build linux

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	webview "github.com/webview/webview_go"
)

//go:embed icon.png
var embeddedIconPNG []byte

const (
	userAgentLinux = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var (
	isAlwaysOnTopLinux = false
	metricsMutex       sync.Mutex
	metricsCacheTime   time.Time
	cachedMetrics      map[string]interface{}
)

func checkSingleInstance() (*os.File, bool) {
	dataDir := getUserDataDir()
	lockPath := filepath.Join(dataDir, "app.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, true
	}
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		// Already running
		return nil, false
	}
	return file, true
}

func getUserDataDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configDir, "WhatsAppDesk")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func ensureAppIconFileLinux(dir string) string {
	iconPath := filepath.Join(dir, "app_icon.png")
	if _, err := os.Stat(iconPath); os.IsNotExist(err) && len(embeddedIconPNG) > 0 {
		_ = os.WriteFile(iconPath, embeddedIconPNG, 0644)
	}
	return iconPath
}

func showNativeNotification(title, message, iconPath string) {
	if iconPath != "" {
		_ = exec.Command("notify-send", "-a", "WhatsApp Desk", "-i", iconPath, title, message).Run()
	} else {
		_ = exec.Command("notify-send", "-a", "WhatsApp Desk", title, message).Run()
	}
}

func toggleAlwaysOnTopLinux() bool {
	isAlwaysOnTopLinux = !isAlwaysOnTopLinux
	state := "remove"
	if isAlwaysOnTopLinux {
		state = "add"
	}
	if path, err := exec.LookPath("wmctrl"); err == nil && path != "" {
		_ = exec.Command("wmctrl", "-r", windowTitle, "-b", fmt.Sprintf("%s,above", state)).Run()
	}
	return isAlwaysOnTopLinux
}

func getAutoStartDesktopPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "autostart", "whatsapp-desk.desktop")
}

func toggleAutoStartLinux() bool {
	p := getAutoStartDesktopPath()
	if p == "" {
		return false
	}
	if _, err := os.Stat(p); err == nil {
		_ = os.Remove(p)
		return false
	}

	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	_ = os.MkdirAll(filepath.Dir(p), 0755)
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=WhatsApp Desk
Comment=Lightweight WhatsApp Desktop Client
Exec=%s
Icon=whatsapp-desk
Terminal=false
Categories=Network;InstantMessaging;
StartupNotify=true
`, execPath)

	err = os.WriteFile(p, []byte(desktopContent), 0644)
	return err == nil
}

func loadWindowState(dir string) *WindowState {
	path := filepath.Join(dir, "window_state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	if state.Width < 450 || state.Height < 320 {
		return nil
	}
	return &state
}

var lastSavedStateLinux *WindowState

func saveWindowState(dir string, width, height int) {
	if width >= 450 && height >= 320 {
		if lastSavedStateLinux != nil &&
			lastSavedStateLinux.Width == float64(width) &&
			lastSavedStateLinux.Height == float64(height) {
			return // Avoid redundant disk writes
		}
		state := WindowState{
			Width:  float64(width),
			Height: float64(height),
		}
		data, err := json.MarshalIndent(state, "", "  ")
		if err == nil {
			_ = os.WriteFile(filepath.Join(dir, "window_state.json"), data, 0644)
			lastSavedStateLinux = &state
		}
	}
}

func runApp() {
	lockFile, isSingle := checkSingleInstance()
	if !isSingle {
		fmt.Println("WhatsApp Desk is already running.")
		os.Exit(0)
	}
	if lockFile != nil {
		defer lockFile.Close()
	}

	userDataDir := getUserDataDir()
	cleanupPreviewDir()
	defer cleanupPreviewDir()

	// Prevent WebKitGTK WebProcess crash on Wayland / Mesa EGL (SkiaGLContext)
	_ = os.Unsetenv("WEBKIT_FORCE_COMPOSITING_MODE")
	if os.Getenv("WEBKIT_DISABLE_COMPOSITING_MODE") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

	// Enable WebKit Remote Inspector for live debugging and DevTools at http://127.0.0.1:9222
	if os.Getenv("WEBKIT_INSPECTOR_SERVER") == "" {
		_ = os.Setenv("WEBKIT_INSPECTOR_SERVER", "127.0.0.1:9222")
	}

	// Restore window state if previously saved
	initialWidth := windowWidth
	initialHeight := windowHeight
	state := loadWindowState(userDataDir)
	if state != nil {
		initialWidth = int(state.Width)
		initialHeight = int(state.Height)
	}

	w := webview.New(true)
	if w == nil {
		log.Fatalln("Gagal inisialisasi WebKitGTK Webview")
	}
	defer w.Destroy()

	w.SetTitle(windowTitle)
	w.SetSize(initialWidth, initialHeight, webview.HintNone)

	// Bind window state saver from JS resize events
	_ = w.Bind("saveWindowStateNative", func(width, height int) {
		saveWindowState(userDataDir, width, height)
	})

	iconPath := ensureAppIconFileLinux(userDataDir)

	// Bind native notification bridge
	_ = w.Bind("sendNativeNotification", func(title, body string) {
		go showNativeNotification(title, body, iconPath)
	})

	// Bind external link handler (xdg-open)
	_ = w.Bind("openExternalLink", func(rawURL string) {
		if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
			go func() {
				_ = exec.Command("xdg-open", rawURL).Start()
			}()
		}
	})

	// Bind dock badge (no-op on linux)
	_ = w.Bind("updateDockBadge", func(badge string) {
		// Not supported on standard Linux window managers
	})

	// Bind Always on Top status & toggle
	_ = w.Bind("isAlwaysOnTopNative", func() bool {
		return isAlwaysOnTopLinux
	})

	_ = w.Bind("toggleAlwaysOnTopNative", func() bool {
		return toggleAlwaysOnTopLinux()
	})

	// Bind Auto-Start status & toggle
	_ = w.Bind("isAutoStartNative", func() bool {
		p := getAutoStartDesktopPath()
		if p == "" {
			return false
		}
		_, err := os.Stat(p)
		return err == nil
	})

	_ = w.Bind("toggleAutoStartNative", func() bool {
		return toggleAutoStartLinux()
	})

	// Bind process metrics for Reverse Engineering HUD (cached for 1.5s to avoid STW pauses)
	_ = w.Bind("getProcessMetricsNative", func() map[string]interface{} {
		metricsMutex.Lock()
		defer metricsMutex.Unlock()
		if time.Since(metricsCacheTime) < 1500*time.Millisecond && cachedMetrics != nil {
			return cachedMetrics
		}
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		cachedMetrics = map[string]interface{}{
			"pid":        os.Getpid(),
			"goroutines": runtime.NumGoroutine(),
			"alloc_mb":   fmt.Sprintf("%.1f MB", float64(m.Alloc)/(1024*1024)),
			"sys_mb":     fmt.Sprintf("%.1f MB", float64(m.Sys)/(1024*1024)),
			"gc_runs":    m.NumGC,
			"inspector":  "http://127.0.0.1:9222",
		}
		metricsCacheTime = time.Now()
		return cachedMetrics
	})

	// Bind download, preview, and settings handlers
	_ = w.Bind("saveDownloadedFileNative", func(filename, dataURI string) string {
		path, err := saveDownloadedFile(filename, dataURI)
		if err != nil {
			return ""
		}
		return path
	})

	_ = w.Bind("previewDocumentNative", func(filename, dataURI string) string {
		path, err := previewDocument(filename, dataURI)
		if err != nil {
			return ""
		}
		return path
	})

	_ = w.Bind("openFileNative", func(filePath string) bool {
		return openFileInDefaultApp(filePath)
	})

	_ = w.Bind("getDownloadDirNative", func() string {
		s := loadSettings()
		return s.DownloadDir
	})

	_ = w.Bind("chooseDownloadDirNative", func() string {
		selected, err := chooseFolderDialog()
		if err != nil || selected == "" {
			return ""
		}
		s := loadSettings()
		s.DownloadDir = selected
		_ = saveSettings(s)
		return selected
	})

	_ = w.Bind("openDownloadDirNative", func() bool {
		s := loadSettings()
		_ = openFolderInFileManager(s.DownloadDir)
		return true
	})

	_ = w.Bind("resetDownloadDirNative", func() string {
		s := loadSettings()
		s.DownloadDir = getDefaultDownloadDir()
		_ = saveSettings(s)
		return s.DownloadDir
	})

	_ = w.Bind("getAppThemeNative", func() string {
		s := loadSettings()
		return s.Theme
	})

	_ = w.Bind("setAppThemeNative", func(theme string) string {
		return saveTheme(theme)
	})

	w.Init(getInitScript(userAgentLinux))
	w.Navigate(appURL)

	defer saveWindowState(userDataDir, initialWidth, initialHeight)
	w.Run()
}
