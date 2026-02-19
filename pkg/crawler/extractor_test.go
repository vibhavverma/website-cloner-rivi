package crawler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtensionDirMapping(t *testing.T) {
	expected := map[string]string{
		// CSS
		".css": "css",
		// JS
		".js": "js",
		// Images
		".jpg": "imgs", ".jpeg": "imgs", ".gif": "imgs", ".png": "imgs", ".svg": "imgs",
		".webp": "imgs", ".avif": "imgs", ".ico": "imgs",
		// Fonts
		".woff": "fonts", ".woff2": "fonts", ".ttf": "fonts", ".eot": "fonts", ".otf": "fonts",
		// Assets
		".json": "assets", ".webmanifest": "assets", ".map": "assets",
		".lottie": "assets",
		// Media
		".mp4": "media", ".webm": "media", ".ogg": "media", ".mp3": "media",
		// 3D Models
		".glb": "models", ".gltf": "models", ".obj": "models",
		".fbx": "models", ".dae": "models",
		// Textures
		".hdr": "textures", ".exr": "textures", ".ktx2": "textures",
		".basis": "textures", ".dds": "textures",
		// Shaders
		".glsl": "shaders", ".vert": "shaders", ".frag": "shaders",
		".vs": "shaders", ".fs": "shaders",
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

func TestIsAssetExtension(t *testing.T) {
	if !IsAssetExtension(".glb") {
		t.Error("expected .glb to be recognized as asset extension")
	}
	if !IsAssetExtension(".css") {
		t.Error("expected .css to be recognized as asset extension")
	}
	if IsAssetExtension(".xyz") {
		t.Error("expected .xyz to NOT be recognized as asset extension")
	}
}

func TestExtensionDir(t *testing.T) {
	if got := ExtensionDir(".glb"); got != "models" {
		t.Errorf("ExtensionDir(.glb) = %q, want %q", got, "models")
	}
	if got := ExtensionDir(".unknown"); got != "" {
		t.Errorf("ExtensionDir(.unknown) = %q, want empty", got)
	}
}

func TestResolveCollision(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("no existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "new.css")
		result := resolveCollision(path, []byte("content"))
		if result != path {
			t.Errorf("expected original path, got %q", result)
		}
	})

	t.Run("same content", func(t *testing.T) {
		path := filepath.Join(tmpDir, "same.css")
		os.WriteFile(path, []byte("same content"), 0644)
		result := resolveCollision(path, []byte("same content"))
		if result != path {
			t.Errorf("expected original path for same content, got %q", result)
		}
	})

	t.Run("different content", func(t *testing.T) {
		path := filepath.Join(tmpDir, "diff.css")
		os.WriteFile(path, []byte("original"), 0644)
		result := resolveCollision(path, []byte("different"))
		if result == path {
			t.Error("expected different path for different content")
		}
		if !strings.HasPrefix(filepath.Base(result), "diff-") {
			t.Errorf("expected hash-suffixed name, got %q", filepath.Base(result))
		}
		if !strings.HasSuffix(result, ".css") {
			t.Errorf("expected .css extension preserved, got %q", result)
		}
	})
}
