package crawler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// HTMLExtractor downloads the HTML content from a URL and saves it to index.html
func HTMLExtractor(link string, projectPath string) error {
	return HTMLExtractorToPath(link, projectPath, "index.html")
}

// HTMLExtractorToPath downloads HTML content and saves it to a specific relative path
func HTMLExtractorToPath(link string, projectPath string, relPath string) error {
	fmt.Println("Extracting HTML from --> ", link)

	resp, err := http.Get(link)
	if err != nil {
		return fmt.Errorf("failed to GET HTML: %w", err)
	}
	defer resp.Body.Close()

	htmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTML body: %w", err)
	}

	fullPath := filepath.Join(projectPath, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
	}
	if err := os.WriteFile(fullPath, htmlData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", relPath, err)
	}

	return nil
}
