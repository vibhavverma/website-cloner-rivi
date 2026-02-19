package crawler

import (
	"testing"
)

func TestExtensionDirMapping(t *testing.T) {
	expected := map[string]string{
		// existing
		".css": "css", ".js": "js",
		".jpg": "imgs", ".jpeg": "imgs", ".gif": "imgs", ".png": "imgs", ".svg": "imgs",
		// new image formats
		".webp": "imgs", ".avif": "imgs", ".ico": "imgs",
		// fonts
		".woff": "fonts", ".woff2": "fonts", ".ttf": "fonts", ".eot": "fonts", ".otf": "fonts",
		// other
		".json": "assets", ".webmanifest": "assets", ".map": "assets",
		// media
		".mp4": "media", ".webm": "media", ".ogg": "media", ".mp3": "media",
	}

	for ext, dir := range expected {
		got, ok := extensionDir[ext]
		if !ok {
			t.Errorf("extension %q not found in extensionDir", ext)
			continue
		}
		if got != dir {
			t.Errorf("extensionDir[%q] = %q, want %q", ext, got, dir)
		}
	}
}
