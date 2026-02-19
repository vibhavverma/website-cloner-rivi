package css

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteURLs(t *testing.T) {
	projectDir := "/project"
	cssFile := "/project/css/style.css"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "background image with double quotes",
			input:    `background: url("https://example.com/hero.jpg");`,
			expected: `background: url("../imgs/hero.jpg");`,
		},
		{
			name:     "background image with single quotes",
			input:    `background: url('https://example.com/bg.png');`,
			expected: `background: url('../imgs/bg.png');`,
		},
		{
			name:     "background image without quotes",
			input:    `background: url(https://example.com/bg.png);`,
			expected: `background: url(../imgs/bg.png);`,
		},
		{
			name:     "font-face src",
			input:    `src: url("https://fonts.example.com/roboto.woff2") format("woff2");`,
			expected: `src: url("../fonts/roboto.woff2") format("woff2");`,
		},
		{
			name:     "data URI preserved",
			input:    `background: url("data:image/png;base64,abc123");`,
			expected: `background: url("data:image/png;base64,abc123");`,
		},
		{
			name:     "already rewritten path preserved",
			input:    `background: url("../imgs/hero.jpg");`,
			expected: `background: url("../imgs/hero.jpg");`,
		},
		{
			name:     "relative URL with extension",
			input:    `background: url("../images/hero.jpg");`,
			expected: `background: url("../imgs/hero.jpg");`,
		},
		{
			name:     "multiple URLs in one rule",
			input:    `background: url("bg.jpg"), url("overlay.png");`,
			expected: `background: url("../imgs/bg.jpg"), url("../imgs/overlay.png");`,
		},
		{
			name:     "unknown extension preserved",
			input:    `something: url("file.xyz");`,
			expected: `something: url("file.xyz");`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RewriteURLs(tt.input, cssFile, projectDir)
			if result != tt.expected {
				t.Errorf("\n  got:  %s\n  want: %s", result, tt.expected)
			}
		})
	}
}

func TestRewriteFile(t *testing.T) {
	// Create temp project structure
	tmpDir := t.TempDir()
	cssDir := filepath.Join(tmpDir, "css")
	os.MkdirAll(cssDir, 0755)

	cssContent := `body {
  background: url("https://example.com/bg.jpg");
}
@font-face {
  src: url("https://fonts.example.com/font.woff2");
}`

	cssPath := filepath.Join(cssDir, "style.css")
	os.WriteFile(cssPath, []byte(cssContent), 0644)

	if err := RewriteFile(cssPath, tmpDir); err != nil {
		t.Fatalf("RewriteFile error: %v", err)
	}

	result, _ := os.ReadFile(cssPath)
	resultStr := string(result)

	if !strings.Contains(resultStr, `url("../imgs/bg.jpg")`) {
		t.Errorf("expected background image to be rewritten, got: %s", resultStr)
	}
	if !strings.Contains(resultStr, `url("../fonts/font.woff2")`) {
		t.Errorf("expected font to be rewritten, got: %s", resultStr)
	}
}

func TestRewriteAllCSS(t *testing.T) {
	tmpDir := t.TempDir()
	cssDir := filepath.Join(tmpDir, "css")
	os.MkdirAll(cssDir, 0755)

	// Write two CSS files
	css1 := `body { background: url("https://example.com/bg1.jpg"); }`
	css2 := `div { background: url("https://example.com/bg2.png"); }`
	os.WriteFile(filepath.Join(cssDir, "a.css"), []byte(css1), 0644)
	os.WriteFile(filepath.Join(cssDir, "b.css"), []byte(css2), 0644)

	if err := RewriteAllCSS(tmpDir); err != nil {
		t.Fatalf("RewriteAllCSS error: %v", err)
	}

	r1, _ := os.ReadFile(filepath.Join(cssDir, "a.css"))
	if !strings.Contains(string(r1), "../imgs/bg1.jpg") {
		t.Errorf("a.css not rewritten: %s", r1)
	}

	r2, _ := os.ReadFile(filepath.Join(cssDir, "b.css"))
	if !strings.Contains(string(r2), "../imgs/bg2.png") {
		t.Errorf("b.css not rewritten: %s", r2)
	}
}
