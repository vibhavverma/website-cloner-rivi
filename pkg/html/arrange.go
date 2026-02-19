package html

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// arrange rewrites resource links in a single HTML file to point to local paths
func arrange(htmlFilePath string, projectDir string) error {
	input, err := os.ReadFile(htmlFilePath)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(input))
	if err != nil {
		return err
	}

	// Calculate relative prefix from this HTML file's location to the project root
	relDir, _ := filepath.Rel(filepath.Dir(htmlFilePath), projectDir)
	if relDir == "" || relDir == "." {
		relDir = ""
	} else {
		relDir += "/"
	}

	// Replace JS links
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("src")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("src", relDir+"js/"+file)
		}
	})

	// Replace CSS links
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("href")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("href", relDir+"css/"+file)
		}
	})

	// Replace IMG links (src)
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("src")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("src", relDir+"imgs/"+file)
		}
	})

	// Replace favicon/icon links
	doc.Find("link[rel='icon'], link[rel='shortcut icon'], link[rel='apple-touch-icon']").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("href")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("href", relDir+"imgs/"+file)
		}
	})

	// Replace preload links
	doc.Find("link[rel='preload']").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("href")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			ext := strings.ToLower(filepath.Ext(file))
			dir := extensionToDir(ext)
			if dir != "" {
				s.SetAttr("href", relDir+dir+"/"+file)
			}
		}
	})

	// Replace video/audio src
	doc.Find("video[src], audio[src]").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("src")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("src", relDir+"media/"+file)
		}
	})

	// Replace video poster
	doc.Find("video[poster]").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("poster")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("poster", relDir+"imgs/"+file)
		}
	})

	// Replace source src
	doc.Find("source[src]").Each(func(i int, s *goquery.Selection) {
		data, exists := s.Attr("src")
		if exists && shouldRewrite(data) {
			file := filepath.Base(data)
			s.SetAttr("src", relDir+"media/"+file)
		}
	})

	html, err := doc.Html()
	if err != nil {
		return err
	}

	return os.WriteFile(htmlFilePath, []byte(html), 0644)
}

// arrangeAll walks the project directory and rewrites links in all HTML files
func arrangeAll(projectDir string) error {
	return filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".html" || ext == ".htm" {
			if err := arrange(path, projectDir); err != nil {
				return err
			}
		}
		return nil
	})
}

// shouldRewrite returns true if the URL should be rewritten to a local path.
// It skips data URIs, blob URIs, and URLs that already point to our local dir structure.
func shouldRewrite(u string) bool {
	if u == "" || strings.HasPrefix(u, "data:") || strings.HasPrefix(u, "blob:") {
		return false
	}
	// Skip URLs already rewritten to local directories
	localDirs := []string{"css/", "js/", "imgs/", "fonts/", "assets/", "media/", "models/", "textures/", "shaders/"}
	for _, d := range localDirs {
		if strings.HasPrefix(u, d) || strings.HasPrefix(u, "./"+d) || strings.HasPrefix(u, "../"+d) {
			return false
		}
	}
	return true
}

// extensionToDir maps a file extension to its local directory
func extensionToDir(ext string) string {
	switch ext {
	case ".css":
		return "css"
	case ".js":
		return "js"
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".avif", ".ico":
		return "imgs"
	case ".woff", ".woff2", ".ttf", ".eot", ".otf":
		return "fonts"
	case ".mp4", ".webm", ".ogg", ".mp3":
		return "media"
	case ".json", ".webmanifest", ".map", ".lottie":
		return "assets"
	case ".glb", ".gltf", ".obj", ".fbx", ".dae":
		return "models"
	case ".hdr", ".exr", ".ktx2", ".basis", ".dds":
		return "textures"
	case ".glsl", ".vert", ".frag", ".vs", ".fs":
		return "shaders"
	default:
		return ""
	}
}
