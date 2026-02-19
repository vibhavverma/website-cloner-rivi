//go:build integration

package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cssRewriter "github.com/goclone-dev/goclone/pkg/css"
	"github.com/goclone-dev/goclone/pkg/file"
	htmlRewriter "github.com/goclone-dev/goclone/pkg/html"
)

// newIntegrationTestServer creates a realistic test website for integration testing
func newIntegrationTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Main page with diverse resource types
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <title>Integration Test</title>
  <link rel="stylesheet" href="/css/style.css">
  <link rel="icon" href="/favicon.ico">
  <link rel="preload" href="/fonts/test.woff2" as="font">
  <script src="/js/app.js"></script>
</head>
<body>
  <h1>Test Page</h1>
  <img src="/images/hero.png" alt="Hero">
  <img srcset="/images/small.jpg 480w, /images/large.jpg 800w" alt="Responsive">
  <a href="/about">About</a>
  <video src="/video/intro.mp4" poster="/images/poster.jpg"></video>
</body>
</html>`))
	})

	// About page
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <title>About</title>
  <link rel="stylesheet" href="/css/style.css">
  <script src="/js/app.js"></script>
</head>
<body>
  <h1>About Page</h1>
  <a href="/">Home</a>
</body>
</html>`))
	})

	// CSS with url() references
	mux.HandleFunc("/css/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(`body {
  background: url("https://example.com/bg.jpg");
  font-family: 'TestFont';
}
@font-face {
  font-family: 'TestFont';
  src: url("https://example.com/font.woff2") format("woff2");
}`))
	})

	// JS
	mux.HandleFunc("/js/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(`console.log('integration test');`))
	})

	// Images
	mux.HandleFunc("/images/hero.png", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PNG_DATA"))
	})
	mux.HandleFunc("/images/small.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("SMALL_JPG"))
	})
	mux.HandleFunc("/images/large.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("LARGE_JPG"))
	})
	mux.HandleFunc("/images/poster.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("POSTER_JPG"))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("FAVICON"))
	})

	// Fonts
	mux.HandleFunc("/fonts/test.woff2", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("WOFF2_DATA"))
	})

	// Video
	mux.HandleFunc("/video/intro.mp4", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("MP4_DATA"))
	})

	return httptest.NewServer(mux)
}

func TestIntegrationFullPipeline(t *testing.T) {
	ts := newIntegrationTestServer()
	defer ts.Close()

	// Create project
	projectPath := file.CreateProject("integration-test")
	defer os.RemoveAll(projectPath)

	// Run crawl
	ctx := context.Background()
	opts := CrawlOptions{
		Depth:    0,
		Parallel: 5,
		Delay:    0,
	}
	if err := Crawl(ctx, ts.URL, projectPath, nil, opts); err != nil {
		t.Fatalf("Crawl failed: %v", err)
	}

	// Run HTML link rewriting
	if err := htmlRewriter.LinkRestructure(projectPath); err != nil {
		t.Fatalf("LinkRestructure failed: %v", err)
	}

	// Run CSS url() rewriting
	if err := cssRewriter.RewriteAllCSS(projectPath); err != nil {
		t.Fatalf("RewriteAllCSS failed: %v", err)
	}

	// Verify files exist
	expectedFiles := []string{
		"index.html",
		"css/style.css",
		"js/app.js",
		"imgs/hero.png",
		"imgs/favicon.ico",
		"fonts/test.woff2",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", f)
		}
	}

	// Verify HTML links were rewritten
	indexContent, err := os.ReadFile(filepath.Join(projectPath, "index.html"))
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	indexStr := string(indexContent)

	if !strings.Contains(indexStr, "css/style.css") {
		t.Error("expected CSS link to be rewritten to css/style.css")
	}
	if !strings.Contains(indexStr, "js/app.js") {
		t.Error("expected JS link to be rewritten to js/app.js")
	}
	if !strings.Contains(indexStr, "imgs/hero.png") {
		t.Error("expected img src to be rewritten to imgs/hero.png")
	}

	// Verify CSS url() was rewritten
	cssContent, err := os.ReadFile(filepath.Join(projectPath, "css", "style.css"))
	if err != nil {
		t.Fatalf("failed to read style.css: %v", err)
	}
	cssStr := string(cssContent)

	if !strings.Contains(cssStr, "../imgs/bg.jpg") {
		t.Errorf("expected background url to be rewritten, got: %s", cssStr)
	}
	if !strings.Contains(cssStr, "../fonts/font.woff2") {
		t.Errorf("expected font url to be rewritten, got: %s", cssStr)
	}
}

func TestIntegrationSPADetection(t *testing.T) {
	// Create an SPA-like test server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><script src="bundle.js"></script><script src="vendor.js"></script><script src="app.js"></script></head>
<body>
  <div id="root"></div>
  <noscript>You need to enable JavaScript to run this app.</noscript>
</body>
</html>`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	score, err := DetectSPA(ts.URL)
	if err != nil {
		t.Fatalf("DetectSPA failed: %v", err)
	}

	if !score.IsSPA() {
		t.Errorf("expected SPA detection to return true, got score=%d, reasons=%v",
			score.Score, score.Reasons)
	}
}
