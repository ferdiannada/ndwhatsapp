package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type UIController interface {
	Dispatch(func())
	Eval(string)
}

const (
	appVersion = "1.5.4"
	githubRepo = "vianziro/Whatsapp-Dekstop"
)

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	Assets  []GitHubAsset `json:"assets"`
}

type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseTitle   string `json:"release_title"`
	DownloadURL    string `json:"download_url"`
	AssetSize      int64  `json:"asset_size"`
}

func parseVersionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		if idx := strings.IndexAny(p, "-+"); idx != -1 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err == nil {
			res = append(res, n)
		} else {
			res = append(res, 0)
		}
	}
	for len(res) < 3 {
		res = append(res, 0)
	}
	return res
}

func isNewerVersion(current, latest string) bool {
	c := parseVersionParts(current)
	l := parseVersionParts(latest)
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

func findAssetForCurrentOS(release *GitHubRelease) *GitHubAsset {
	return findAssetForOS(release, runtime.GOOS)
}

func findAssetForOS(release *GitHubRelease, goos string) *GitHubAsset {
	if goos == "darwin" {
		for _, a := range release.Assets {
			if strings.HasSuffix(strings.ToLower(a.Name), ".zip") {
				return &a
			}
		}
	} else if goos == "windows" {
		for _, a := range release.Assets {
			if strings.HasSuffix(strings.ToLower(a.Name), ".exe") {
				return &a
			}
		}
	} else if goos == "linux" {
		for _, a := range release.Assets {
			name := strings.ToLower(a.Name)
			if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") {
				return &a
			}
		}
	}
	return nil
}

func updateDownloadExtension(downloadURL string) string {
	parsed, err := url.Parse(downloadURL)
	if err == nil {
		path := strings.ToLower(parsed.Path)
		if strings.HasSuffix(path, ".tar.gz") {
			return ".tar.gz"
		}
		if ext := filepath.Ext(path); ext == ".zip" || ext == ".exe" || ext == ".tgz" {
			return ext
		}
	}
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	if runtime.GOOS == "linux" {
		return ".tar.gz"
	}
	return ".zip"
}

func checkForUpdate(currentVer string) (*UpdateInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "WhatsAppDesktopLight-Updater")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	latestVer := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	info := &UpdateInfo{
		CurrentVersion: currentVer,
		LatestVersion:  latestVer,
		ReleaseTitle:   rel.Name,
	}

	if isNewerVersion(currentVer, latestVer) {
		asset := findAssetForCurrentOS(&rel)
		if asset != nil {
			info.Available = true
			info.DownloadURL = asset.BrowserDownloadURL
			info.AssetSize = asset.Size
		}
	}

	return info, nil
}

type progressWriter struct {
	total      int64
	downloaded int64
	lastPct    int
	onProgress func(percent int)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	if pw.total > 0 && pw.onProgress != nil {
		pct := int((pw.downloaded * 100) / pw.total)
		if pct != pw.lastPct {
			pw.lastPct = pct
			pw.onProgress(pct)
		}
	}
	return n, nil
}

func downloadFileWithProgress(url, destPath string, onProgress func(int)) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "WhatsAppDesktopLight-Updater")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		total:      resp.ContentLength,
		onProgress: onProgress,
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
	return err
}

func executeUpdate(ui UIController, downloadURL string) error {
	ext := updateDownloadExtension(downloadURL)
	destFile := filepath.Join(os.TempDir(), "whatsapp_update_download"+ext)
	_ = os.Remove(destFile)

	ui.Dispatch(func() {
		ui.Eval("if (window.onUpdateStatus) { window.onUpdateStatus('Mengunduh pembaruan... 0%'); }")
	})

	err := downloadFileWithProgress(downloadURL, destFile, func(pct int) {
		ui.Dispatch(func() {
			ui.Eval(fmt.Sprintf("if (window.onUpdateProgress) { window.onUpdateProgress(%d); }", pct))
		})
	})
	if err != nil {
		ui.Dispatch(func() {
			ui.Eval(fmt.Sprintf("if (window.onUpdateError) { window.onUpdateError(%q); }", err.Error()))
		})
		return err
	}

	ui.Dispatch(func() {
		ui.Eval("if (window.onUpdateStatus) { window.onUpdateStatus('Memasang pembaruan & memulai ulang...'); }")
	})

	time.Sleep(600 * time.Millisecond)
	return applyUpdate(destFile)
}
