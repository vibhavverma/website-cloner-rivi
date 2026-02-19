package crawler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/goclone-dev/goclone/pkg/parser"
)

// file extension map for directing files to their proper directory in O(1) time
var extensionDir = map[string]string{
	// CSS
	".css": "css",
	// JS
	".js": "js",
	// Images
	".jpg": "imgs", ".jpeg": "imgs", ".gif": "imgs",
	".png": "imgs", ".svg": "imgs", ".webp": "imgs",
	".avif": "imgs", ".ico": "imgs",
	// Fonts
	".woff": "fonts", ".woff2": "fonts", ".ttf": "fonts",
	".eot": "fonts", ".otf": "fonts",
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

// IsAssetExtension returns true if the extension is a known asset type
func IsAssetExtension(ext string) bool {
	_, ok := extensionDir[ext]
	return ok
}

// ExtensionDir returns the directory name for a given extension, or empty string
func ExtensionDir(ext string) string {
	return extensionDir[ext]
}

// Extractor downloads a resource from link and saves it to the appropriate directory
func Extractor(link string, projectPath string) error {
	fmt.Println("Extracting --> ", link)

	resp, err := http.Get(link)
	if err != nil {
		return fmt.Errorf("failed to GET %s: %w", link, err)
	}
	defer resp.Body.Close()

	base := parser.URLFilename(link)
	ext := parser.URLExtension(link)

	if ext != "" {
		dirPath := extensionDir[ext]
		if dirPath != "" {
			if err := writeFileToPath(projectPath, base, dirPath, resp); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFileToPath(projectPath, filename, fileDir string, resp *http.Response) error {
	fullPath := filepath.Join(projectPath, fileDir, filename)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", fullPath, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle filename collisions: if file exists with different content, use hash suffix
	finalPath := resolveCollision(fullPath, data)

	if err := os.WriteFile(finalPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", finalPath, err)
	}
	return nil
}

// resolveCollision checks if a file already exists at the given path.
// If it exists with the same content, returns the same path (no-op write).
// If it exists with different content, returns a new path with a hash suffix.
// If it doesn't exist, returns the original path.
func resolveCollision(fullPath string, newContent []byte) string {
	existing, err := os.ReadFile(fullPath)
	if err != nil {
		// File doesn't exist, use original path
		return fullPath
	}

	// File exists — check if content is identical
	if bytes.Equal(existing, newContent) {
		return fullPath
	}

	// Content differs — create a unique name with content hash
	ext := filepath.Ext(fullPath)
	base := strings.TrimSuffix(filepath.Base(fullPath), ext)
	dir := filepath.Dir(fullPath)

	hash := sha256.Sum256(newContent)
	hashSuffix := fmt.Sprintf("%x", hash[:4]) // 8-char hex

	newName := fmt.Sprintf("%s-%s%s", base, hashSuffix, ext)
	return filepath.Join(dir, newName)
}
