package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type AppSettings struct {
	DownloadDir      string `json:"download_dir"`
	NotifyOnDownload bool   `json:"notify_on_download"`
	Theme            string `json:"theme"` // "dark", "light", "system"
}

var (
	settingsMutex  sync.RWMutex
	cachedSettings *AppSettings
)

func getDefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Downloads", "WhatsApp Downloads")
}

func getSettingsFilePath() string {
	var baseDir string
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, "Library", "Application Support", "ndwhatsapp")
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = os.Getenv("APPDATA")
			if configDir == "" {
				configDir = "."
			}
		}
		baseDir = filepath.Join(configDir, "ndwhatsapp")
	}
	_ = os.MkdirAll(baseDir, 0755)
	return filepath.Join(baseDir, "settings.json")
}

func loadSettings() *AppSettings {
	settingsMutex.RLock()
	if cachedSettings != nil {
		defer settingsMutex.RUnlock()
		clone := *cachedSettings
		return &clone
	}
	settingsMutex.RUnlock()

	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if cachedSettings != nil {
		clone := *cachedSettings
		return &clone
	}

	s := &AppSettings{
		DownloadDir:      getDefaultDownloadDir(),
		NotifyOnDownload: true,
		Theme:            "dark",
	}
	data, err := os.ReadFile(getSettingsFilePath())
	if err == nil {
		_ = json.Unmarshal(data, s)
	}
	if strings.TrimSpace(s.DownloadDir) == "" {
		s.DownloadDir = getDefaultDownloadDir()
	}
	if strings.TrimSpace(s.Theme) == "" {
		s.Theme = "dark"
	}
	cachedSettings = s
	clone := *s
	return &clone
}

func saveSettings(s *AppSettings) error {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	targetPath := getSettingsFilePath()
	tmpPath := targetPath + fmt.Sprintf(".tmp.%d", os.Getpid())
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	cachedSettings = s
	return nil
}

func saveTheme(theme string) string {
	if theme != "dark" && theme != "light" && theme != "system" {
		theme = "dark"
	}
	s := loadSettings()
	s.Theme = theme
	_ = saveSettings(s)
	return s.Theme
}

func saveDownloadedFile(filename, dataURI string) (string, error) {
	settings := loadSettings()
	return saveDownloadedFileToDir(settings.DownloadDir, filename, dataURI)
}

func saveDownloadedFileToDir(targetDir, filename, dataURI string) (string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori tujuan: %w", err)
	}

	// Sanitize filename against directory traversal
	filename = filepath.Base(filepath.Clean(filename))
	if filename == "." || filename == "/" || filename == "" {
		filename = "download"
	}

	// Extract base64 payload
	var payload string
	idx := strings.Index(dataURI, ";base64,")
	if idx != -1 {
		payload = dataURI[idx+8:]
	} else {
		payload = dataURI
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(payload))

	// Atomically create target file using O_CREATE|O_EXCL to eliminate TOCTOU race
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if base == "" {
		base = "download"
	}

	var f *os.File
	var targetPath string
	for counter := 0; ; counter++ {
		if counter == 0 {
			targetPath = filepath.Join(targetDir, filename)
		} else {
			targetPath = filepath.Join(targetDir, fmt.Sprintf("%s (%d)%s", base, counter, ext))
		}
		var err error
		f, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("gagal membuat berkas tujuan: %w", err)
		}
	}
	defer f.Close()

	// Stream decoder directly into file - zero 50MB+ Go slice allocation
	if _, err := io.Copy(f, decoder); err != nil {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("gagal decode/menulis data: %w", err)
	}

	return targetPath, nil
}

func openFolderInFileManager(folderPath string) error {
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		_ = os.MkdirAll(folderPath, 0755)
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", folderPath).Start()
	} else if runtime.GOOS == "windows" {
		return exec.Command("explorer.exe", folderPath).Start()
	} else if runtime.GOOS == "linux" {
		return exec.Command("xdg-open", folderPath).Start()
	}
	return nil
}

var safePreviewExtensions = map[string]bool{
	".pdf":  true,
	".txt":  true,
	".csv":  true,
	".rtf":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,
}

func isSafePreviewExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return safePreviewExtensions[ext]
}

func getPreviewDir() string {
	return filepath.Join(os.TempDir(), "ndwhatsapp-preview")
}

func previewDocument(filename, dataURI string) (string, error) {
	if !isSafePreviewExtension(filename) {
		return "", fmt.Errorf("tipe berkas tidak diizinkan untuk pratinjau otomatis demi keamanan: %s", filepath.Ext(filename))
	}

	tempDir := getPreviewDir()
	_ = os.MkdirAll(tempDir, 0700)
	_ = os.Chmod(tempDir, 0700)

	targetPath, err := saveDownloadedFileToDir(tempDir, filename, dataURI)
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", targetPath)
	default:
		cmd = exec.Command("xdg-open", targetPath)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
	return targetPath, nil
}

func openFileInDefaultApp(filePath string) bool {
	if filePath == "" {
		return false
	}
	if !isSafePreviewExtension(filePath) {
		return false
	}
	if _, err := os.Stat(filePath); err != nil {
		return false
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	default:
		cmd = exec.Command("xdg-open", filePath)
	}
	if cmd != nil {
		_ = cmd.Start()
		return true
	}
	return false
}

func cleanupPreviewDir() {
	tempDir := getPreviewDir()
	_ = os.RemoveAll(tempDir)
}

