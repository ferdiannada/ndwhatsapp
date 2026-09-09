//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var folderDialogLock sync.Mutex

func chooseFolderDialog() (string, error) {
	if !folderDialogLock.TryLock() {
		return "", nil
	}
	defer folderDialogLock.Unlock()

	// Try zenity (GNOME / Ubuntu / standard GTK desktops)
	if path, err := exec.LookPath("zenity"); err == nil && path != "" {
		out, err := exec.Command("zenity", "--file-selection", "--directory", "--title=Pilih Folder Penyimpanan File WhatsApp").Output()
		if err == nil {
			res := strings.TrimSpace(string(out))
			if res != "" {
				return res, nil
			}
		}
	}

	// Try kdialog (KDE Plasma desktops)
	if path, err := exec.LookPath("kdialog"); err == nil && path != "" {
		home, _ := os.UserHomeDir()
		out, err := exec.Command("kdialog", "--getexistingdirectory", home, "--title", "Pilih Folder Penyimpanan File WhatsApp").Output()
		if err == nil {
			res := strings.TrimSpace(string(out))
			if res != "" {
				return res, nil
			}
		}
	}

	// Fallback to user Downloads folder
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	downloads := filepath.Join(home, "Downloads")
	_ = os.MkdirAll(downloads, 0755)
	return downloads, nil
}
