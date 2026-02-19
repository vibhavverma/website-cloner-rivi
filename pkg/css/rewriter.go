package css

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// urlPattern matches CSS url() references, capturing the quote and URL inside
// Handles: url("..."), url('...'), url(...)
var urlPattern = regexp.MustCompile(`url\(\s*(['"]?)(.*?)['"]?\s*\)`)

// RewriteFile reads a CSS file, rewrites url() references to local paths,
// and writes the result back
func RewriteFile(cssFilePath string, projectDir string) error {
	content, err := os.ReadFile(cssFilePath)
	if err != nil {
		return err
	}

	rewritten := RewriteURLs(string(content), cssFilePath, projectDir)

	return os.WriteFile(cssFilePath, []byte(rewritten), 0644)
}

// RewriteURLs replaces url() references in CSS content with local paths
func RewriteURLs(content string, cssFilePath string, projectDir string) string {
	// Calculate relative path from CSS file location to project root
	cssDir := filepath.Dir(cssFilePath)
	relToRoot, _ := filepath.Rel(cssDir, projectDir)
	if relToRoot == "" || relToRoot == "." {
		relToRoot = ""
	} else {
		relToRoot += "/"
	}

	return urlPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := urlPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}

		quote := sub[1]
		rawURL := sub[2]

		// Skip data URIs, blob URIs, and empty URLs
		if rawURL == "" || strings.HasPrefix(rawURL, "data:") || strings.HasPrefix(rawURL, "blob:") || strings.HasPrefix(rawURL, "#") {
			return match
		}

		// Skip URLs already pointing to local directories
		localDirs := []string{"css/", "js/", "imgs/", "fonts/", "assets/", "media/", "models/", "textures/", "shaders/"}
		for _, d := range localDirs {
			if strings.HasPrefix(rawURL, d) || strings.HasPrefix(rawURL, "./"+d) || strings.HasPrefix(rawURL, "../"+d) {
				return match
			}
		}

		// Extract filename and determine target directory
		filename := filepath.Base(rawURL)
		ext := strings.ToLower(filepath.Ext(filename))
		dir := extensionToDir(ext)
		if dir == "" {
			return match
		}

		newURL := relToRoot + dir + "/" + filename
		return "url(" + quote + newURL + quote + ")"
	})
}

// RewriteAllCSS walks the project directory and rewrites url() in all CSS files
func RewriteAllCSS(projectDir string) error {
	return filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".css" {
			if err := RewriteFile(path, projectDir); err != nil {
				return err
			}
		}
		return nil
	})
}

// extensionToDir maps a file extension to its local directory
func extensionToDir(ext string) string {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".avif", ".ico":
		return "imgs"
	case ".woff", ".woff2", ".ttf", ".eot", ".otf":
		return "fonts"
	case ".mp4", ".webm", ".ogg", ".mp3":
		return "media"
	case ".glb", ".gltf", ".obj", ".fbx", ".dae":
		return "models"
	case ".hdr", ".exr", ".ktx2", ".basis", ".dds":
		return "textures"
	case ".cur":
		return "imgs"
	default:
		return ""
	}
}
