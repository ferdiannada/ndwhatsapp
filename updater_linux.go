//go:build linux

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func extractTarGz(srcFile, destDir string) error {
	f, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	_ = os.MkdirAll(destDir, 0755)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(targetPath, 0755)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func applyUpdateLinux(downloadedFile string) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	newBinaryPath := downloadedFile
	if strings.HasSuffix(strings.ToLower(downloadedFile), ".tar.gz") || strings.HasSuffix(strings.ToLower(downloadedFile), ".tgz") {
		extractDir := filepath.Join(os.TempDir(), "whatsapp_linux_extracted")
		_ = os.RemoveAll(extractDir)
		if err := extractTarGz(downloadedFile, extractDir); err != nil {
			return fmt.Errorf("tar extract error: %w", err)
		}

		// Find binary in extracted directory
		entries, _ := os.ReadDir(extractDir)
		for _, e := range entries {
			if !e.IsDir() && (strings.Contains(strings.ToLower(e.Name()), "whatsapp") || e.Name() == "whatsapp-desk" || e.Name() == "whatsapp-desktop-light") {
				newBinaryPath = filepath.Join(extractDir, e.Name())
				break
			}
		}
	}

	if _, err := os.Stat(newBinaryPath); err != nil {
		return fmt.Errorf("new binary not found: %w", err)
	}

	// Verify that the new binary's dependencies are satisfied on this Linux distro
	if lddOut, err := exec.Command("ldd", newBinaryPath).CombinedOutput(); err == nil {
		if strings.Contains(string(lddOut), "not found") {
			return fmt.Errorf("pembaruan tidak kompatibel dengan sistem Anda (pustaka tidak ditemukan / webkit mismatch)")
		}
	}

	// Prepare background swap and restart command
	script := fmt.Sprintf(`(sleep 1 && cp %q %q && chmod +x %q && %q) &`,
		newBinaryPath, execPath, execPath, execPath)

	cmd := exec.Command("sh", "-c", script)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start restart script: %w", err)
	}

	os.Exit(0)
	return nil
}

func applyUpdate(downloadedFile string) error {
	return applyUpdateLinux(downloadedFile)
}
