package crawler

import (
	"testing"
)

func TestShouldCaptureAsset(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/models/scene.glb", true},
		{"https://example.com/textures/wood.hdr", true},
		{"https://example.com/style.css", true},
		{"https://example.com/app.js", true},
		{"https://example.com/font.woff2", true},
		{"https://example.com/image.png", true},
		{"https://example.com/shader.glsl", true},
		{"https://example.com/texture.ktx2", true},
		{"https://example.com/page", false},     // no extension
		{"https://example.com/api/data", false},  // no recognized extension
		{"data:image/png;base64,abc", false},      // data URI
		{"blob:https://example.com/abc", false},   // blob URI
		{"", false},
	}

	for _, tt := range tests {
		got := shouldCaptureAsset(tt.url)
		if got != tt.expected {
			t.Errorf("shouldCaptureAsset(%q) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestUrlToLocalPath(t *testing.T) {
	projectPath := "/tmp/project"

	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/assets/model.glb", "/tmp/project/assets/model.glb"},
		{"https://cdn.example.com/textures/wood.jpg", "/tmp/project/textures/wood.jpg"},
		{"https://example.com/js/app.js", "/tmp/project/js/app.js"},
		{"https://example.com/deep/path/to/file.css", "/tmp/project/deep/path/to/file.css"},
		{"", ""},
	}

	for _, tt := range tests {
		got := urlToLocalPath(tt.url, projectPath)
		if got != tt.expected {
			t.Errorf("urlToLocalPath(%q) = %q, want %q", tt.url, got, tt.expected)
		}
	}
}

func TestUrlToLocalPathSecurity(t *testing.T) {
	projectPath := "/tmp/project"

	// Directory traversal attempts should be blocked
	malicious := []string{
		"https://example.com/../../../etc/passwd",
		"https://example.com/assets/../../secret",
	}

	for _, u := range malicious {
		got := urlToLocalPath(u, projectPath)
		if got != "" {
			t.Errorf("urlToLocalPath(%q) should return empty for traversal, got %q", u, got)
		}
	}
}
