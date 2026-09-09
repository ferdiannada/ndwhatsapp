package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type AppSettings struct {
	DownloadDir      string `json:"download_dir"`
	NotifyOnDownload bool   `json:"notify_on_download"`
	Theme            string `json:"theme"` // "dark", "light", "system"
}

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
		baseDir = filepath.Join(home, "Library", "Application Support", "WhatsAppDesktopLight")
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = os.Getenv("APPDATA")
			if configDir == "" {
				configDir = "."
			}
		}
		baseDir = filepath.Join(configDir, "WhatsAppDesktopLight")
	}
	_ = os.MkdirAll(baseDir, 0755)
	return filepath.Join(baseDir, "settings.json")
}

func loadSettings() *AppSettings {
	s := &AppSettings{
		DownloadDir:      getDefaultDownloadDir(),
		NotifyOnDownload: true,
		Theme:            "dark",
	}
	data, err := os.ReadFile(getSettingsFilePath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, s)
	if strings.TrimSpace(s.DownloadDir) == "" {
		s.DownloadDir = getDefaultDownloadDir()
	}
	if strings.TrimSpace(s.Theme) == "" {
		s.Theme = "dark"
	}
	return s
}

func saveSettings(s *AppSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getSettingsFilePath(), data, 0644)
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

func getUniqueFilePath(dir, filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if base == "" {
		base = "download"
	}
	target := filepath.Join(dir, filename)
	counter := 1
	for {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			return target
		}
		target = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, counter, ext))
		counter++
	}
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

	// Extract and decode base64 payload
	var rawBytes []byte
	var err error
	idx := strings.Index(dataURI, ";base64,")
	if idx != -1 {
		rawBytes, err = base64.StdEncoding.DecodeString(dataURI[idx+8:])
	} else {
		rawBytes, err = base64.StdEncoding.DecodeString(dataURI)
	}
	if err != nil {
		return "", fmt.Errorf("gagal decode base64: %w", err)
	}

	targetPath := getUniqueFilePath(targetDir, filename)
	if err := os.WriteFile(targetPath, rawBytes, 0644); err != nil {
		return "", fmt.Errorf("gagal menyimpan berkas: %w", err)
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

func previewDocument(filename, dataURI string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), "WhatsAppDeskPreview")
	_ = os.MkdirAll(tempDir, 0755)

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

