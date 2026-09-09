package main

import (
	"encoding/base64"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestSecuritySafePreviewExtensions(t *testing.T) {
	safeFiles := []string{
		"invoice.pdf",
		"REPORT.PDF",
		"photo.png",
		"avatar.JPG",
		"document.docx",
		"notes.txt",
		"data.csv",
		"sheet.xlsx",
		"slides.pptx",
	}

	for _, f := range safeFiles {
		if !isSafePreviewExtension(f) {
			t.Errorf("expected %s to be recognized as safe for preview, but was rejected", f)
		}
	}

	dangerousFiles := []string{
		"malware.exe",
		"script.sh",
		"startup.desktop",
		"payload.bat",
		"macro.vbs",
		"hack.py",
		"exploit.bin",
		"app.apk",
		"code.js",
		"test.cmd",
		"binary",
		"",
	}

	for _, f := range dangerousFiles {
		if isSafePreviewExtension(f) {
			t.Errorf("expected %s to be REJECTED for preview, but was allowed", f)
		}
	}
}

func TestSecuritySaveDownloadedFileToDirTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ndwa-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	payload := "data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte("hello world"))

	// Test path traversal attempt
	maliciousFilename := "../../../../evil.txt"
	savedPath, err := saveDownloadedFileToDir(tempDir, maliciousFilename, payload)
	if err != nil {
		t.Fatalf("unexpected error saving file: %v", err)
	}

	// Must be contained strictly within tempDir and have base name "evil.txt"
	expectedPath := filepath.Join(tempDir, "evil.txt")
	if savedPath != expectedPath {
		t.Errorf("path traversal not sanitized properly! got %s, expected %s", savedPath, expectedPath)
	}

	// Verify file was written
	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(content))
	}
}

func TestSecurityExternalLinkValidation(t *testing.T) {
	validURLs := []string{
		"https://web.whatsapp.com",
		"http://example.com/page?test=1",
		"https://faq.whatsapp.com/123",
	}

	for _, raw := range validURLs {
		u, err := url.ParseRequestURI(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			t.Errorf("expected valid HTTP/HTTPS URL %s, but rejected", raw)
		}
	}

	invalidURLs := []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"gopher://evil.com",
		"/bin/bash",
		"http://",
		"",
	}

	for _, raw := range invalidURLs {
		u, err := url.ParseRequestURI(raw)
		isValid := err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
		if isValid {
			t.Errorf("expected invalid/dangerous URL %s to be rejected, but passed", raw)
		}
	}
}
