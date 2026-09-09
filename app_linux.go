//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <math.h>

static gdouble last_pinch_scale = 1.0;

static gboolean on_webview_touchpad_event(GtkWidget *widget, GdkEvent *event, gpointer user_data) {
	(void)widget;
	if (event->type == GDK_TOUCHPAD_PINCH) {
		GdkEventTouchpadPinch *pinch = (GdkEventTouchpadPinch *)event;
		WebKitWebView *view = WEBKIT_WEB_VIEW(user_data);

		if (pinch->phase == GDK_TOUCHPAD_GESTURE_PHASE_BEGIN) {
			last_pinch_scale = 1.0;
		} else if (pinch->phase == GDK_TOUCHPAD_GESTURE_PHASE_UPDATE) {
			gdouble delta_scale = pinch->scale - last_pinch_scale;
			if (fabs(delta_scale) >= 0.015) {
				last_pinch_scale = pinch->scale;
				if (view) {
					char js[160];
					snprintf(js, sizeof(js),
						"window.__onNativeTouchpadPinch && window.__onNativeTouchpadPinch(%f, %f, %f);",
						delta_scale, pinch->x, pinch->y);
					webkit_web_view_evaluate_javascript(view, js, -1, NULL, NULL, NULL, NULL, NULL);
				}
			}
		} else if (pinch->phase == GDK_TOUCHPAD_GESTURE_PHASE_END || pinch->phase == GDK_TOUCHPAD_GESTURE_PHASE_CANCEL) {
			last_pinch_scale = 1.0;
		}

		return TRUE; // Suppress native WebKitGTK whole-window zoom
	}
	return FALSE;
}

static void on_webview_zoom_level_notify(WebKitWebView *web_view, GParamSpec *pspec, gpointer user_data) {
	(void)pspec;
	(void)user_data;
	if (webkit_web_view_get_zoom_level(web_view) != 1.0) {
		webkit_web_view_set_zoom_level(web_view, 1.0);
	}
}

static WebKitWebView* find_webkit_view(GtkWidget *widget) {
	if (!widget) return NULL;
	if (WEBKIT_IS_WEB_VIEW(widget)) {
		return WEBKIT_WEB_VIEW(widget);
	}
	if (GTK_IS_BIN(widget)) {
		return find_webkit_view(gtk_bin_get_child(GTK_BIN(widget)));
	}
	if (GTK_IS_CONTAINER(widget)) {
		GList *children = gtk_container_get_children(GTK_CONTAINER(widget));
		for (GList *iter = children; iter != NULL; iter = g_list_next(iter)) {
			WebKitWebView *found = find_webkit_view(GTK_WIDGET(iter->data));
			if (found) {
				g_list_free(children);
				return found;
			}
		}
		g_list_free(children);
	}
	return NULL;
}

static void attach_touchpad_pinch_filter(void *gtk_window_ptr) {
	if (!gtk_window_ptr) return;
	GtkWidget *window = GTK_WIDGET(gtk_window_ptr);
	WebKitWebView *view = find_webkit_view(window);
	if (view) {
		GtkWidget *view_widget = GTK_WIDGET(view);
		// Intercept native touchpad pinch before WebKitGTK processes it
		g_signal_connect(view_widget, "event", G_CALLBACK(on_webview_touchpad_event), view);
		g_signal_connect(window, "event", G_CALLBACK(on_webview_touchpad_event), view);
		// Secondary lock: ensure WebKit zoom-level is always locked to 1.0
		g_signal_connect(view, "notify::zoom-level", G_CALLBACK(on_webview_zoom_level_notify), NULL);
		webkit_web_view_set_zoom_level(view, 1.0);
	}
}
*/
import "C"

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	dir := filepath.Join(configDir, "ndwhatsapp")
	_ = os.MkdirAll(dir, 0700)
	_ = os.Chmod(dir, 0700)
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
	return filepath.Join(home, ".config", "autostart", "ndwhatsapp.desktop")
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
Name=ndWhatsApp
Comment=ndWhatsApp - Native Client & Reverse Engineering Suite
Exec=%s
Icon=ndwhatsapp
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

	// Prevent WebKitGTK WebProcess crash on Wayland / Mesa EGL (SkiaGLContext / __eglFini)
	_ = os.Unsetenv("WEBKIT_FORCE_COMPOSITING_MODE")
	if os.Getenv("WEBKIT_DISABLE_COMPOSITING_MODE") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

	// Enable WebKit Remote Inspector only in debug mode
	if debugMode {
		if os.Getenv("WEBKIT_INSPECTOR_SERVER") == "" {
			_ = os.Setenv("WEBKIT_INSPECTOR_SERVER", "127.0.0.1:9222")
		}
		log.Println("⚠️ [SECURITY] Debug mode aktif: WebKit Remote Inspector berjalan di http://127.0.0.1:9222")
	} else {
		_ = os.Unsetenv("WEBKIT_INSPECTOR_SERVER")
	}

	// Restore window state if previously saved
	initialWidth := windowWidth
	initialHeight := windowHeight
	state := loadWindowState(userDataDir)
	if state != nil {
		initialWidth = int(state.Width)
		initialHeight = int(state.Height)
	}

	w := webview.New(debugMode)
	if w == nil {
		log.Fatalln("Gagal inisialisasi WebKitGTK Webview")
	}
	defer w.Destroy()

	w.SetTitle(windowTitle)
	w.SetSize(initialWidth, initialHeight, webview.HintNone)

	// Intercept touchpad pinch gestures at GTK level to prevent full-window zooming & stuttering
	C.attach_touchpad_pinch_filter(w.Window())

	// Bind window state saver from JS resize events
	_ = w.Bind("saveWindowStateNative", func(width, height int) {
		saveWindowState(userDataDir, width, height)
	})

	iconPath := ensureAppIconFileLinux(userDataDir)

	// Bind native notification bridge
	_ = w.Bind("sendNativeNotification", func(title, body string) {
		go showNativeNotification(title, body, iconPath)
	})

	// Bind external link handler (xdg-open) with strict scheme validation
	_ = w.Bind("openExternalLink", func(rawURL string) {
		u, err := url.ParseRequestURI(rawURL)
		if err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			go func() {
				_ = exec.Command("xdg-open", u.String()).Start()
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
		inspURL := "disabled (jalankan dengan --debug)"
		if debugMode {
			inspURL = "http://127.0.0.1:9222"
		}
		cachedMetrics = map[string]interface{}{
			"pid":        os.Getpid(),
			"goroutines": runtime.NumGoroutine(),
			"alloc_mb":   fmt.Sprintf("%.1f MB", float64(m.Alloc)/(1024*1024)),
			"sys_mb":     fmt.Sprintf("%.1f MB", float64(m.Sys)/(1024*1024)),
			"gc_runs":    m.NumGC,
			"inspector":  inspURL,
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
