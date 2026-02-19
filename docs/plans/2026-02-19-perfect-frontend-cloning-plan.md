# Perfect Frontend Cloning — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make goclone produce visually identical local clones of any website — static or SPA — by fixing resource discovery, adding multi-page crawling, CSS rewriting, and headless browser fallback.

**Architecture:** Enhance the existing Colly-based crawler with extended resource discovery (25+ file types, srcset, CSS url()), add BFS multi-page crawling with depth control, and integrate chromedp as an automatic fallback when SPA-like pages are detected. The link rewriting pipeline is expanded to process all HTML files and CSS files.

**Tech Stack:** Go 1.24, Colly v2, goquery, chromedp (new), regexp for CSS url() parsing

---

## Task 1: Extend File Extension Map

**Files:**
- Modify: `pkg/crawler/extractor.go:14-23`
- Create: `pkg/crawler/extractor_test.go`

**Step 1: Write the failing test**

Create `pkg/crawler/extractor_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestExtensionDirMapping -v`
Expected: FAIL — missing `.webp`, `.woff2`, etc.

**Step 3: Expand the extension map in `extractor.go`**

Replace `extensionDir` (lines 14-23) with:

```go
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
	// Media
	".mp4": "media", ".webm": "media", ".ogg": "media", ".mp3": "media",
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestExtensionDirMapping -v`
Expected: PASS

**Step 5: Update `pkg/file/write.go` to create new directories**

Add `fonts`, `assets`, `media` directory creation in `CreateProject()`. Replace lines 22-24:

```go
for _, dir := range []string{"css", "js", "imgs", "fonts", "assets", "media"} {
	err := os.MkdirAll(filepath.Join(projectPath, dir), 0755)
	check(err)
}
```

Also fix all `0777` permissions to `0755` (dirs) and `0644` (files) in `write.go`.

**Step 6: Fix file permissions in `extractor.go`**

Change `0777` to `0755` on line 66 (`MkdirAll`) and `0644` on line 72 (`OpenFile`).

**Step 7: Commit**

```bash
git add pkg/crawler/extractor.go pkg/crawler/extractor_test.go pkg/file/write.go
git commit -m "feat: extend file type support to 25+ extensions

Add fonts (.woff/.woff2/.ttf/.eot/.otf), modern images (.webp/.avif/.ico),
media (.mp4/.webm/.ogg/.mp3), and assets (.json/.webmanifest/.map).
Fix 0777 permissions to 0755/0644."
```

---

## Task 2: Fix Security Issues (TLS, Panic, Permissions)

**Files:**
- Modify: `pkg/crawler/html.go:15`
- Modify: `pkg/parser/url.go:62-64`

**Step 1: Write failing test for GetDomain error handling**

Create `pkg/parser/url_error_test.go`:

```go
package parser

import "testing"

func TestGetDomainInvalidURL(t *testing.T) {
	_, err := GetDomainSafe("://not-a-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestGetDomainValidURL(t *testing.T) {
	domain, err := GetDomainSafe("https://example.com/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain != "example.com" {
		t.Fatalf("expected example.com, got %s", domain)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/parser/ -run TestGetDomain -v`
Expected: FAIL — `GetDomainSafe` not defined

**Step 3: Add `GetDomainSafe` and fix `HTMLExtractor`**

In `pkg/parser/url.go`, add below `GetDomain`:

```go
// GetDomainSafe is like GetDomain but returns an error instead of panicking
func GetDomainSafe(validurl string) (string, error) {
	u, err := url.Parse(validurl)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", validurl, err)
	}
	return u.Hostname(), nil
}
```

Add `"fmt"` to imports.

In `pkg/crawler/html.go`, replace the global TLS hack (line 15) with a local client:

```go
func HTMLExtractor(link string, projectPath string) error {
	fmt.Println("Extracting HTML from --> ", link)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(link)
	if err != nil {
		return fmt.Errorf("failed to GET HTML: %v", err)
	}
	defer resp.Body.Close()

	htmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTML body: %v", err)
	}

	filePath := filepath.Join(projectPath, "index.html")
	return os.WriteFile(filePath, htmlData, 0644)
}
```

Update imports: add `"time"`, `"path/filepath"`, remove `"crypto/tls"`.

**Step 4: Update callers of GetDomain**

In `cmd/clone.go:93`, replace `parser.GetDomain(u)` with:

```go
var err error
name, err = parser.GetDomainSafe(u)
if err != nil {
	return "", fmt.Errorf("invalid URL %q: %w", u, err)
}
```

**Step 5: Run all tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./... -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add pkg/crawler/html.go pkg/parser/url.go pkg/parser/url_error_test.go cmd/clone.go
git commit -m "fix: remove global TLS skip, add GetDomainSafe, fix permissions

Replace InsecureSkipVerify global hack with proper HTTP client.
Add GetDomainSafe that returns error instead of panicking.
Use filepath.Join and 0644 permissions."
```

---

## Task 3: Enhanced Resource Discovery in Collector

**Files:**
- Modify: `pkg/crawler/collector.go:27-64`
- Modify: `testutils/utils.go`
- Modify: `pkg/crawler/crawler_test.go`

**Step 1: Update test server with new resource types**

In `testutils/utils.go`, add new test content and handlers:

```go
var CrawlerFontContent = "wOFF2fake"
var CrawlerFaviconContent = "iconfake"

// Add to NewCrawlerTestServer() mux:
// Font handler
mux.HandleFunc("/font.woff2", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "font/woff2")
	w.WriteHeader(200)
	w.Write([]byte(CrawlerFontContent))
})

// Favicon handler
mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(CrawlerFaviconContent))
})
```

Update `CrawlerIndexContent` to include new tags:

```go
var CrawlerIndexContent = `<html>
	<link rel="stylesheet" href="index.css">
	<link rel="icon" href="favicon.ico">
	<link rel="preload" href="font.woff2" as="font" type="font/woff2">
	<script src="index.js"></script>
	<img src="image.png" alt="Red dot" />
	<img srcset="image.png 1x, image.png 2x" />
</html>`
```

**Step 2: Write test for font download**

In `pkg/crawler/crawler_test.go`, add to `collectorTests`:

```go
"fontDownload": func(t *testing.T) {
	projectDirectory := file.CreateProject("test")
	fontContent := collectAndGetFileContent(TsUrl, projectDirectory, "/fonts/font.woff2")
	if fontContent != testutils.CrawlerFontContent {
		t.Fatalf("Expect %q, got: %q", testutils.CrawlerFontContent, fontContent)
	}
	os.RemoveAll(projectDirectory)
},
"faviconDownload": func(t *testing.T) {
	projectDirectory := file.CreateProject("test")
	content := collectAndGetFileContent(TsUrl, projectDirectory, "/imgs/favicon.ico")
	if content != testutils.CrawlerFaviconContent {
		t.Fatalf("Expect %q, got: %q", testutils.CrawlerFaviconContent, content)
	}
	os.RemoveAll(projectDirectory)
},
```

**Step 3: Run tests to verify they fail**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestCollectorTests -v`
Expected: FAIL — fonts not discovered

**Step 4: Extend collector.go with new OnHTML handlers**

Add after the existing `img[src]` handler (after line 64):

```go
// Favicon and preload links
c.OnHTML("link[rel='icon'], link[rel='shortcut icon'], link[rel='apple-touch-icon']", func(e *colly.HTMLElement) {
	link := e.Attr("href")
	if link == "" {
		return
	}
	fmt.Println("Favicon found", "-->", link)
	if err := Extractor(e.Request.AbsoluteURL(link), projectPath); err != nil {
		fmt.Printf("warning: failed to extract %s: %v\n", link, err)
	}
})

// Preload resources (fonts, critical CSS/JS)
c.OnHTML("link[rel='preload']", func(e *colly.HTMLElement) {
	link := e.Attr("href")
	if link == "" {
		return
	}
	fmt.Println("Preload found", "-->", link)
	if err := Extractor(e.Request.AbsoluteURL(link), projectPath); err != nil {
		fmt.Printf("warning: failed to extract %s: %v\n", link, err)
	}
})

// Responsive images (srcset)
c.OnHTML("img[srcset], source[srcset]", func(e *colly.HTMLElement) {
	srcset := e.Attr("srcset")
	for _, entry := range parseSrcset(srcset) {
		fmt.Println("Srcset found", "-->", entry)
		if err := Extractor(e.Request.AbsoluteURL(entry), projectPath); err != nil {
			fmt.Printf("warning: failed to extract %s: %v\n", entry, err)
		}
	}
})

// Video and audio sources
c.OnHTML("video[src], audio[src], source[src]", func(e *colly.HTMLElement) {
	link := e.Attr("src")
	if link == "" || strings.HasPrefix(link, "data:") || strings.HasPrefix(link, "blob:") {
		return
	}
	fmt.Println("Media found", "-->", link)
	if err := Extractor(e.Request.AbsoluteURL(link), projectPath); err != nil {
		fmt.Printf("warning: failed to extract %s: %v\n", link, err)
	}
})
```

Add the `parseSrcset` helper at the bottom of `collector.go`:

```go
// parseSrcset parses an HTML srcset attribute value into individual URLs.
// e.g. "img-300.jpg 300w, img-600.jpg 600w" → ["img-300.jpg", "img-600.jpg"]
func parseSrcset(srcset string) []string {
	var urls []string
	for _, part := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) >= 1 {
			urls = append(urls, fields[0])
		}
	}
	return urls
}
```

**Step 5: Run tests to verify they pass**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add pkg/crawler/collector.go testutils/utils.go pkg/crawler/crawler_test.go
git commit -m "feat: discover favicons, preload resources, srcset, media

Add OnHTML handlers for link[rel=icon], link[rel=preload],
img[srcset], source[srcset], video[src], audio[src].
Add parseSrcset helper."
```

---

## Task 4: Multi-Page BFS Crawl Queue

**Files:**
- Create: `pkg/crawler/queue.go`
- Create: `pkg/crawler/queue_test.go`

**Step 1: Write failing test for the page queue**

Create `pkg/crawler/queue_test.go`:

```go
package crawler

import "testing"

func TestPageQueue_AddAndNext(t *testing.T) {
	q := NewPageQueue(3)
	q.Add("https://example.com", 0)
	q.Add("https://example.com/about", 1)

	url, depth, ok := q.Next()
	if !ok {
		t.Fatal("expected item from queue")
	}
	if url != "https://example.com" || depth != 0 {
		t.Fatalf("got url=%s depth=%d", url, depth)
	}

	url, depth, ok = q.Next()
	if !ok {
		t.Fatal("expected item from queue")
	}
	if url != "https://example.com/about" || depth != 1 {
		t.Fatalf("got url=%s depth=%d", url, depth)
	}

	_, _, ok = q.Next()
	if ok {
		t.Fatal("expected empty queue")
	}
}

func TestPageQueue_Dedup(t *testing.T) {
	q := NewPageQueue(10)
	q.Add("https://example.com", 0)
	q.Add("https://example.com", 0) // duplicate

	q.Next() // first
	_, _, ok := q.Next()
	if ok {
		t.Fatal("expected dedup to prevent second entry")
	}
}

func TestPageQueue_DepthLimit(t *testing.T) {
	q := NewPageQueue(1)
	q.Add("https://example.com", 0)
	q.Add("https://example.com/deep", 2) // beyond depth limit

	q.Next() // depth 0
	_, _, ok := q.Next()
	if ok {
		t.Fatal("expected depth limit to block deep URL")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestPageQueue -v`
Expected: FAIL — `NewPageQueue` not defined

**Step 3: Implement the queue**

Create `pkg/crawler/queue.go`:

```go
package crawler

import "sync"

type pageEntry struct {
	url   string
	depth int
}

// PageQueue is a BFS queue for crawling pages with depth limiting and dedup.
type PageQueue struct {
	mu       sync.Mutex
	items    []pageEntry
	visited  map[string]bool
	maxDepth int // -1 means unlimited
}

// NewPageQueue creates a new PageQueue. maxDepth -1 means unlimited.
func NewPageQueue(maxDepth int) *PageQueue {
	return &PageQueue{
		visited:  make(map[string]bool),
		maxDepth: maxDepth,
	}
}

// Add enqueues a URL if it hasn't been visited and is within depth limit.
func (q *PageQueue) Add(url string, depth int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.visited[url] {
		return
	}
	if q.maxDepth >= 0 && depth > q.maxDepth {
		return
	}
	q.visited[url] = true
	q.items = append(q.items, pageEntry{url: url, depth: depth})
}

// Next returns the next URL to process. Returns false if queue is empty.
func (q *PageQueue) Next() (string, int, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return "", 0, false
	}
	entry := q.items[0]
	q.items = q.items[1:]
	return entry.url, entry.depth, true
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestPageQueue -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/crawler/queue.go pkg/crawler/queue_test.go
git commit -m "feat: add BFS page queue with depth limiting and dedup"
```

---

## Task 5: Multi-Page Crawling Integration

**Files:**
- Modify: `pkg/crawler/crawler.go`
- Modify: `pkg/crawler/collector.go`
- Modify: `pkg/crawler/html.go`
- Modify: `cmd/clone.go`
- Modify: `cmd/root.go`

**Step 1: Add `--depth` flag to CLI**

In `cmd/root.go`, add a new variable and flag (after line 19):

```go
var Depth int
```

In `Execute()`, add to the persistent flags section (after line 69):

```go
pf.IntVarP(&Depth, "depth", "d", -1, "Max crawl depth (-1 for unlimited)")
```

Add `Depth` to `CloneOptions` in `cmd/clone.go`:

```go
type CloneOptions struct {
	Serve     bool
	Open      bool
	ServePort int
	Cookies   []string
	Proxy     string
	UserAgent string
	Depth     int
}
```

And in root.go's Run func, set `Depth: Depth` in the opts struct.

**Step 2: Modify Collector to return discovered same-domain links**

In `pkg/crawler/collector.go`, change `Collector` signature to also return discovered page links:

```go
func Collector(ctx context.Context, url string, projectPath string, cookieJar *cookiejar.Jar, proxyString string, userAgent string) ([]string, error) {
```

Add a `discoveredPages` slice and an `<a href>` handler:

```go
var mu sync.Mutex
var discoveredPages []string

// Discover same-domain page links
c.OnHTML("a[href]", func(e *colly.HTMLElement) {
	link := e.Attr("href")
	absURL := e.Request.AbsoluteURL(link)
	if absURL == "" {
		return
	}
	// Only follow same-domain links
	linkURL, err := neturl.Parse(absURL)
	if err != nil {
		return
	}
	baseURL, err := neturl.Parse(url)
	if err != nil {
		return
	}
	if linkURL.Host == baseURL.Host {
		mu.Lock()
		discoveredPages = append(discoveredPages, absURL)
		mu.Unlock()
	}
})
```

Return `discoveredPages` at the end. Add `net/url` import aliased as `neturl` and `"sync"`.

**Step 3: Modify Crawl to do BFS multi-page crawling**

Rewrite `pkg/crawler/crawler.go`:

```go
package crawler

import (
	"context"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Crawl performs BFS crawling of a site, downloading pages and their assets.
func Crawl(ctx context.Context, site string, projectPath string, cookieJar *cookiejar.Jar, proxyString string, userAgent string, maxDepth int) error {
	queue := NewPageQueue(maxDepth)
	queue.Add(site, 0)

	for {
		pageURL, depth, ok := queue.Next()
		if !ok {
			break
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		// Determine HTML file path for this page
		htmlPath, err := pageToFilePath(site, pageURL, projectPath)
		if err != nil {
			fmt.Printf("warning: skipping %s: %v\n", pageURL, err)
			continue
		}

		// Create directory for the HTML file
		if err := os.MkdirAll(filepath.Dir(htmlPath), 0755); err != nil {
			return fmt.Errorf("failed to create dir for %s: %w", htmlPath, err)
		}

		// Download the HTML
		fmt.Printf("Downloading page (depth %d): %s\n", depth, pageURL)
		if err := HTMLExtractorTo(pageURL, htmlPath); err != nil {
			fmt.Printf("warning: failed to download %s: %v\n", pageURL, err)
			continue
		}

		// Collect assets and discover links
		discovered, err := Collector(ctx, pageURL, projectPath, cookieJar, proxyString, userAgent)
		if err != nil {
			fmt.Printf("warning: failed to collect assets for %s: %v\n", pageURL, err)
		}

		// Add discovered pages to queue
		for _, link := range discovered {
			queue.Add(link, depth+1)
		}
	}

	return nil
}

// pageToFilePath converts a page URL to a local file path.
// e.g. https://example.com/about → projectPath/about/index.html
func pageToFilePath(baseURL, pageURL, projectPath string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	page, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}

	// Get relative path from base
	relPath := strings.TrimPrefix(page.Path, "/")
	if relPath == "" || relPath == base.Path {
		return filepath.Join(projectPath, "index.html"), nil
	}

	// Remove trailing slash
	relPath = strings.TrimSuffix(relPath, "/")

	// If path already has an extension, use it directly
	if filepath.Ext(relPath) != "" {
		return filepath.Join(projectPath, relPath), nil
	}

	// Otherwise treat as directory with index.html
	return filepath.Join(projectPath, relPath, "index.html"), nil
}
```

**Step 4: Add `HTMLExtractorTo` in `html.go`**

Add a new function that saves HTML to a specific path (not just index.html):

```go
// HTMLExtractorTo downloads HTML from a URL and saves it to the specified file path.
func HTMLExtractorTo(link string, filePath string) error {
	fmt.Println("Extracting HTML from --> ", link)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(link)
	if err != nil {
		return fmt.Errorf("failed to GET HTML: %v", err)
	}
	defer resp.Body.Close()

	htmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTML body: %v", err)
	}

	return os.WriteFile(filePath, htmlData, 0644)
}
```

**Step 5: Update `clone.go` to pass `Depth` to `Crawl`**

In `cmd/clone.go:101`, update the Crawl call:

```go
if err := crawler.Crawl(ctx, u, projectPath, jar, opts.Proxy, opts.UserAgent, opts.Depth); err != nil {
```

**Step 6: Run all tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./... -v`
Expected: All PASS (update existing crawler_test.go to match new Collector signature — add `_` for returned pages)

**Step 7: Commit**

```bash
git add pkg/crawler/crawler.go pkg/crawler/collector.go pkg/crawler/html.go cmd/clone.go cmd/root.go
git commit -m "feat: add multi-page BFS crawling with --depth flag

Crawl discovers same-domain <a> links and follows them BFS.
Pages saved with URL-path-based directory structure.
--depth flag controls max crawl depth (-1 = unlimited)."
```

---

## Task 6: Multi-File Link Rewriting

**Files:**
- Modify: `pkg/html/arrange.go`
- Modify: `pkg/html/sorter.go`
- Modify: `pkg/html/arrange_test.go`

**Step 1: Write failing test for multi-file rewriting**

Add to `pkg/html/arrange_test.go`:

```go
func TestArrangeMultipleHTMLFiles(t *testing.T) {
	testutils.SilenceStdoutInTests()

	// Create project with multiple HTML files
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "css"), 0755)
	os.MkdirAll(filepath.Join(dir, "js"), 0755)
	os.MkdirAll(filepath.Join(dir, "about"), 0755)

	indexHTML := `<html><link rel="stylesheet" href="https://example.com/style.css"><a href="/about">About</a></html>`
	aboutHTML := `<html><link rel="stylesheet" href="https://example.com/style.css"><script src="https://example.com/app.js"></script></html>`

	os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexHTML), 0644)
	os.WriteFile(filepath.Join(dir, "about", "index.html"), []byte(aboutHTML), 0644)
	os.WriteFile(filepath.Join(dir, "css", "style.css"), []byte("body{}"), 0644)
	os.WriteFile(filepath.Join(dir, "js", "app.js"), []byte("//js"), 0644)

	if err := LinkRestructure(dir); err != nil {
		t.Fatalf("Error: %v", err)
	}

	// Check about/index.html was also rewritten
	aboutContent := file.GetFileContent(filepath.Join(dir, "about", "index.html"))
	if !strings.Contains(aboutContent, "../css/style.css") {
		t.Fatalf("Expected ../css/style.css in about page, got: %s", aboutContent)
	}
	if !strings.Contains(aboutContent, "../js/app.js") {
		t.Fatalf("Expected ../js/app.js in about page, got: %s", aboutContent)
	}
}
```

Add imports: `"os"`, `"path/filepath"`.

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/html/ -run TestArrangeMultipleHTMLFiles -v`
Expected: FAIL — only index.html is processed

**Step 3: Rewrite `arrange.go` to process all HTML files**

Replace the entire `arrange` function:

```go
package html

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func arrange(projectDir string) error {
	// Find all HTML files in the project
	var htmlFiles []string
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			htmlFiles = append(htmlFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, htmlFile := range htmlFiles {
		if err := rewriteHTMLFile(htmlFile, projectDir); err != nil {
			return err
		}
	}
	return nil
}

func rewriteHTMLFile(htmlFile, projectDir string) error {
	input, err := os.ReadFile(htmlFile)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(input))
	if err != nil {
		return err
	}

	// Compute relative prefix from this HTML file to the project root
	htmlDir := filepath.Dir(htmlFile)
	relToRoot, err := filepath.Rel(htmlDir, projectDir)
	if err != nil {
		relToRoot = "."
	}

	// Rewrite JS links
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("src"); exists {
			file := filepath.Base(data)
			s.SetAttr("src", filepath.Join(relToRoot, "js", file))
		}
	})

	// Rewrite CSS links
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("href"); exists {
			file := filepath.Base(data)
			s.SetAttr("href", filepath.Join(relToRoot, "css", file))
		}
	})

	// Rewrite IMG links
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("src"); exists {
			if strings.HasPrefix(data, "data:") {
				return
			}
			file := filepath.Base(data)
			s.SetAttr("src", filepath.Join(relToRoot, "imgs", file))
		}
	})

	// Rewrite srcset attributes
	doc.Find("img[srcset], source[srcset]").Each(func(i int, s *goquery.Selection) {
		if srcset, exists := s.Attr("srcset"); exists {
			s.SetAttr("srcset", rewriteSrcset(srcset, relToRoot))
		}
	})

	// Rewrite favicon/icon links
	doc.Find("link[rel='icon'], link[rel='shortcut icon'], link[rel='apple-touch-icon']").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("href"); exists {
			file := filepath.Base(data)
			s.SetAttr("href", filepath.Join(relToRoot, "imgs", file))
		}
	})

	// Rewrite preload font links
	doc.Find("link[rel='preload']").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("href"); exists {
			file := filepath.Base(data)
			ext := filepath.Ext(file)
			dir := extensionToDir(ext)
			if dir != "" {
				s.SetAttr("href", filepath.Join(relToRoot, dir, file))
			}
		}
	})

	// Rewrite video/audio sources
	doc.Find("video[src], audio[src], source[src]").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("src"); exists {
			if strings.HasPrefix(data, "data:") || strings.HasPrefix(data, "blob:") {
				return
			}
			file := filepath.Base(data)
			s.SetAttr("src", filepath.Join(relToRoot, "media", file))
		}
	})

	html, err := doc.Html()
	if err != nil {
		return err
	}

	return os.WriteFile(htmlFile, []byte(html), 0644)
}

// rewriteSrcset parses a srcset attribute and rewrites each URL.
func rewriteSrcset(srcset, relToRoot string) string {
	var parts []string
	for _, entry := range strings.Split(srcset, ",") {
		entry = strings.TrimSpace(entry)
		fields := strings.Fields(entry)
		if len(fields) >= 1 {
			file := filepath.Base(fields[0])
			fields[0] = filepath.Join(relToRoot, "imgs", file)
		}
		parts = append(parts, strings.Join(fields, " "))
	}
	return strings.Join(parts, ", ")
}

// extensionToDir maps a file extension to its directory name.
func extensionToDir(ext string) string {
	switch ext {
	case ".css":
		return "css"
	case ".js":
		return "js"
	case ".jpg", ".jpeg", ".gif", ".png", ".svg", ".webp", ".avif", ".ico":
		return "imgs"
	case ".woff", ".woff2", ".ttf", ".eot", ".otf":
		return "fonts"
	case ".mp4", ".webm", ".ogg", ".mp3":
		return "media"
	default:
		return "assets"
	}
}
```

**Step 4: Run tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/html/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/html/arrange.go pkg/html/arrange_test.go
git commit -m "feat: rewrite links in all HTML files with relative paths

Process all .html files (not just index.html).
Compute relative path from each HTML file to project root.
Handle srcset, favicons, preload, video/audio sources."
```

---

## Task 7: CSS url() Rewriting

**Files:**
- Create: `pkg/css/rewriter.go`
- Create: `pkg/css/rewriter_test.go`

**Step 1: Write failing test for CSS url() extraction**

Create `pkg/css/rewriter_test.go`:

```go
package css

import "testing"

func TestExtractURLs(t *testing.T) {
	css := `
body { background: url("../images/hero.jpg"); }
@font-face { src: url('/fonts/Inter.woff2') format('woff2'); }
@import url("reset.css");
div { background-image: url(sprite.png); }
`
	urls := ExtractURLs(css)
	expected := []string{"../images/hero.jpg", "/fonts/Inter.woff2", "reset.css", "sprite.png"}
	if len(urls) != len(expected) {
		t.Fatalf("expected %d URLs, got %d: %v", len(expected), len(urls), urls)
	}
	for i, u := range urls {
		if u != expected[i] {
			t.Errorf("url[%d] = %q, want %q", i, u, expected[i])
		}
	}
}

func TestExtractURLsSkipsDataURIs(t *testing.T) {
	css := `body { background: url("data:image/png;base64,abc"); }`
	urls := ExtractURLs(css)
	if len(urls) != 0 {
		t.Fatalf("expected 0 URLs for data URI, got %d: %v", len(urls), urls)
	}
}

func TestRewriteURLs(t *testing.T) {
	css := `body { background: url("https://example.com/images/hero.jpg"); }`
	rewritten := RewriteURLs(css, func(rawURL string) string {
		return "../imgs/hero.jpg"
	})
	expected := `body { background: url("../imgs/hero.jpg"); }`
	if rewritten != expected {
		t.Fatalf("got %q, want %q", rewritten, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/css/ -v`
Expected: FAIL — package doesn't exist

**Step 3: Implement CSS rewriter**

Create `pkg/css/rewriter.go`:

```go
package css

import (
	"regexp"
	"strings"
)

// urlPattern matches CSS url() references.
// Handles: url("path"), url('path'), url(path)
var urlPattern = regexp.MustCompile(`url\(\s*(['"]?)(.+?)\1\s*\)`)

// ExtractURLs returns all URLs referenced in CSS url() declarations.
// Skips data: URIs and empty URLs.
func ExtractURLs(cssContent string) []string {
	matches := urlPattern.FindAllStringSubmatch(cssContent, -1)
	var urls []string
	for _, m := range matches {
		if len(m) >= 3 {
			u := strings.TrimSpace(m[2])
			if u == "" || strings.HasPrefix(u, "data:") || strings.HasPrefix(u, "blob:") {
				continue
			}
			urls = append(urls, u)
		}
	}
	return urls
}

// RewriteURLs replaces all url() references in CSS using the provided rewrite function.
// The rewrite function receives the raw URL and should return the new URL.
func RewriteURLs(cssContent string, rewrite func(rawURL string) string) string {
	return urlPattern.ReplaceAllStringFunc(cssContent, func(match string) string {
		sub := urlPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		rawURL := strings.TrimSpace(sub[2])
		if rawURL == "" || strings.HasPrefix(rawURL, "data:") || strings.HasPrefix(rawURL, "blob:") {
			return match
		}
		newURL := rewrite(rawURL)
		return `url("` + newURL + `")`
	})
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/css/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/css/rewriter.go pkg/css/rewriter_test.go
git commit -m "feat: add CSS url() extraction and rewriting

Regex-based parser for url() references in CSS.
Handles quoted/unquoted URLs, skips data: URIs.
RewriteURLs applies a rewrite function to each url()."
```

---

## Task 8: Integrate CSS Rewriting Into Pipeline

**Files:**
- Modify: `pkg/html/arrange.go`
- Modify: `pkg/html/arrange_test.go`

**Step 1: Write failing test**

Add to `pkg/html/arrange_test.go`:

```go
func TestArrangeRewritesCSSURLs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "css"), 0755)
	os.MkdirAll(filepath.Join(dir, "imgs"), 0755)
	os.MkdirAll(filepath.Join(dir, "fonts"), 0755)

	indexHTML := `<html><link rel="stylesheet" href="https://example.com/style.css"></html>`
	cssContent := `body { background: url("https://example.com/hero.jpg"); }
@font-face { src: url("https://example.com/Inter.woff2"); }`

	os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexHTML), 0644)
	os.WriteFile(filepath.Join(dir, "css", "style.css"), []byte(cssContent), 0644)
	os.WriteFile(filepath.Join(dir, "imgs", "hero.jpg"), []byte("jpg"), 0644)
	os.WriteFile(filepath.Join(dir, "fonts", "Inter.woff2"), []byte("woff2"), 0644)

	if err := LinkRestructure(dir); err != nil {
		t.Fatalf("Error: %v", err)
	}

	rewrittenCSS := file.GetFileContent(filepath.Join(dir, "css", "style.css"))
	if !strings.Contains(rewrittenCSS, "../imgs/hero.jpg") {
		t.Fatalf("Expected CSS to contain ../imgs/hero.jpg, got: %s", rewrittenCSS)
	}
	if !strings.Contains(rewrittenCSS, "../fonts/Inter.woff2") {
		t.Fatalf("Expected CSS to contain ../fonts/Inter.woff2, got: %s", rewrittenCSS)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/html/ -run TestArrangeRewritesCSSURLs -v`
Expected: FAIL — CSS files not rewritten

**Step 3: Add CSS rewriting to `LinkRestructure`**

In `pkg/html/sorter.go`, expand `LinkRestructure`:

```go
package html

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goclone-dev/goclone/pkg/css"
)

// LinkRestructure rewrites links in all HTML and CSS files in the project directory.
func LinkRestructure(projectDir string) error {
	// Rewrite HTML files
	if err := arrange(projectDir); err != nil {
		return err
	}
	// Rewrite CSS files
	return rewriteCSSFiles(projectDir)
}

func rewriteCSSFiles(projectDir string) error {
	return filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".css") {
			return nil
		}
		return rewriteCSSFile(path, projectDir)
	})
}

func rewriteCSSFile(cssFile, projectDir string) error {
	content, err := os.ReadFile(cssFile)
	if err != nil {
		return err
	}

	cssDir := filepath.Dir(cssFile)

	rewritten := css.RewriteURLs(string(content), func(rawURL string) string {
		// Extract just the filename
		filename := filepath.Base(rawURL)
		ext := filepath.Ext(filename)
		dir := extensionToDir(ext)
		if dir == "" {
			return rawURL
		}
		// Compute relative path from CSS file to the asset
		assetPath := filepath.Join(projectDir, dir, filename)
		rel, err := filepath.Rel(cssDir, assetPath)
		if err != nil {
			fmt.Printf("warning: could not compute relative path for %s: %v\n", rawURL, err)
			return rawURL
		}
		return rel
	})

	return os.WriteFile(cssFile, []byte(rewritten), 0644)
}
```

Note: `extensionToDir` is already defined in `arrange.go` from Task 6.

**Step 4: Run test**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/html/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/html/sorter.go pkg/html/arrange.go pkg/html/arrange_test.go
git commit -m "feat: rewrite CSS url() references to local paths

Walk all .css files, extract url() references, compute
relative paths from each CSS file to the asset directories."
```

---

## Task 9: SPA Detection

**Files:**
- Create: `pkg/crawler/detector.go`
- Create: `pkg/crawler/detector_test.go`

**Step 1: Write failing test**

Create `pkg/crawler/detector_test.go`:

```go
package crawler

import "testing"

func TestDetectSPA_ReactApp(t *testing.T) {
	html := `<html><head></head><body><div id="root"></div><script src="/bundle.js"></script></body></html>`
	if !IsSPA(html) {
		t.Fatal("expected React-like page to be detected as SPA")
	}
}

func TestDetectSPA_StaticSite(t *testing.T) {
	html := `<html><body><h1>Welcome</h1><p>` + string(make([]byte, 600)) + `</p></body></html>`
	if IsSPA(html) {
		t.Fatal("expected content-rich page to NOT be detected as SPA")
	}
}

func TestDetectSPA_NextJS(t *testing.T) {
	html := `<html><body><div id="__next"></div><noscript>You need to enable JavaScript to run this app.</noscript></body></html>`
	if !IsSPA(html) {
		t.Fatal("expected Next.js-like page to be detected as SPA")
	}
}

func TestDetectSPA_EmptyBody(t *testing.T) {
	html := `<html><body><div id="app"></div></body></html>`
	if !IsSPA(html) {
		t.Fatal("expected empty Vue-like page to be detected as SPA")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestDetectSPA -v`
Expected: FAIL — `IsSPA` not defined

**Step 3: Implement detector**

Create `pkg/crawler/detector.go`:

```go
package crawler

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// IsSPA uses a scoring heuristic to detect if HTML content is a Single Page Application.
// Score >= 3 is considered SPA.
func IsSPA(htmlContent string) bool {
	return SPAScore(htmlContent) >= 3
}

// SPAScore computes a heuristic score for how likely the HTML is from a SPA.
func SPAScore(htmlContent string) int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return 0
	}

	score := 0

	// +2: Body has a single child div with common SPA root IDs
	spaIDs := []string{"root", "app", "__next", "__nuxt", "__svelte"}
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		if id, exists := s.Attr("id"); exists {
			for _, spaID := range spaIDs {
				if id == spaID {
					score += 2
					return
				}
			}
		}
	})

	// +2: Body has very little visible text (empty shell)
	bodyText := strings.TrimSpace(doc.Find("body").Text())
	if len(bodyText) < 100 {
		score += 2
	}

	// +1: Script tags with bundle/chunk in filename
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if strings.Contains(src, "bundle") || strings.Contains(src, "chunk") {
			score += 1
		}
	})

	// +1: <noscript> with "enable JavaScript" text
	doc.Find("noscript").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		if strings.Contains(text, "javascript") || strings.Contains(text, "enable") {
			score += 1
		}
	})

	// +1: Meta with framework indicators
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		content, _ := s.Attr("content")
		name, _ := s.Attr("name")
		combined := strings.ToLower(content + " " + name)
		for _, fw := range []string{"react", "vue", "angular", "next", "nuxt", "svelte"} {
			if strings.Contains(combined, fw) {
				score += 1
				return
			}
		}
	})

	// -2: Body has substantial visible text (server-rendered)
	if len(bodyText) > 500 {
		score -= 2
	}

	return score
}
```

**Step 4: Run test**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestDetectSPA -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/crawler/detector.go pkg/crawler/detector_test.go
git commit -m "feat: add SPA detection heuristic

Score-based detection checks for empty body, SPA root IDs,
bundle scripts, noscript tags, and framework meta tags."
```

---

## Task 10: Headless Browser Mode (chromedp)

**Files:**
- Create: `pkg/crawler/headless.go`
- Create: `pkg/crawler/headless_test.go`
- Modify: `go.mod`

**Step 1: Add chromedp dependency**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go get github.com/chromedp/chromedp`

**Step 2: Write failing test**

Create `pkg/crawler/headless_test.go`:

```go
package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadlessRender(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping headless test in CI (no Chrome)")
	}

	// Serve a minimal SPA that renders via JS
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`<html><body><div id="root"></div><script>
			document.getElementById('root').innerHTML = '<h1>Rendered by JS</h1>';
		</script></body></html>`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "index.html")

	err := HeadlessRender(context.Background(), ts.URL, outFile, "", 10)
	if err != nil {
		t.Fatalf("HeadlessRender failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if !strings.Contains(string(content), "Rendered by JS") {
		t.Fatalf("expected rendered JS content, got: %s", string(content))
	}
}
```

**Step 3: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestHeadlessRender -v`
Expected: FAIL — `HeadlessRender` not defined

**Step 4: Implement headless renderer**

Create `pkg/crawler/headless.go`:

```go
package crawler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// HeadlessRender uses Chrome DevTools Protocol to render a page with JS execution
// and saves the fully rendered HTML. waitForSelector is optional (empty = just wait for idle).
// timeoutSecs controls max wait time.
func HeadlessRender(ctx context.Context, url, outputPath, waitForSelector string, timeoutSecs int) error {
	// Check if Chrome is available
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeout := time.Duration(timeoutSecs) * time.Second
	taskCtx, cancel = context.WithTimeout(taskCtx, timeout)
	defer cancel()

	var renderedHTML string

	actions := []chromedp.Action{
		chromedp.Navigate(url),
	}

	if waitForSelector != "" {
		actions = append(actions, chromedp.WaitVisible(waitForSelector, chromedp.ByQuery))
	} else {
		// Wait for network to be idle
		actions = append(actions, chromedp.Sleep(2*time.Second))
	}

	// Get the rendered outer HTML
	actions = append(actions, chromedp.OuterHTML("html", &renderedHTML))

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return fmt.Errorf("chromedp render failed for %s: %w", url, err)
	}

	// Wrap in proper HTML tags if needed
	fullHTML := "<!DOCTYPE html><html>" + renderedHTML + "</html>"
	if err := os.WriteFile(outputPath, []byte(fullHTML), 0644); err != nil {
		return fmt.Errorf("failed to write rendered HTML: %w", err)
	}

	fmt.Printf("Headless render complete: %s\n", url)
	return nil
}

// IsHeadlessAvailable checks if chromedp can launch Chrome.
func IsHeadlessAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	err := chromedp.Run(taskCtx, chromedp.Navigate("about:blank"))
	return err == nil
}
```

**Step 5: Run test**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestHeadlessRender -v`
Expected: PASS (if Chrome is installed locally)

**Step 6: Commit**

```bash
git add pkg/crawler/headless.go pkg/crawler/headless_test.go go.mod go.sum
git commit -m "feat: add headless browser rendering via chromedp

HeadlessRender launches Chrome, navigates, waits for idle/selector,
captures fully rendered DOM. IsHeadlessAvailable for graceful fallback."
```

---

## Task 11: Integrate SPA Detection Into Crawl Pipeline

**Files:**
- Modify: `pkg/crawler/crawler.go`
- Modify: `cmd/clone.go`
- Modify: `cmd/root.go`

**Step 1: Add headless flags**

In `cmd/root.go`, add new variables:

```go
var (
	Headless    bool
	NoHeadless  bool
	WaitFor     string
	WaitTimeout int
	// ... existing vars
)
```

Add flags in `Execute()`:

```go
pf.BoolVar(&Headless, "headless", false, "Force headless Chrome mode for all pages")
pf.BoolVar(&NoHeadless, "no-headless", false, "Disable headless mode entirely")
pf.StringVar(&WaitFor, "wait-for", "", "CSS selector to wait for in headless mode")
pf.IntVar(&WaitTimeout, "wait-timeout", 30, "Max wait time in seconds for headless render")
```

Add to `CloneOptions`:

```go
type CloneOptions struct {
	Serve       bool
	Open        bool
	ServePort   int
	Cookies     []string
	Proxy       string
	UserAgent   string
	Depth       int
	Headless    bool
	NoHeadless  bool
	WaitFor     string
	WaitTimeout int
}
```

**Step 2: Integrate SPA detection into Crawl**

Modify `pkg/crawler/crawler.go` — after downloading HTML, check if it's a SPA:

```go
// After HTMLExtractorTo succeeds, read the HTML and check for SPA
htmlContent, err := os.ReadFile(htmlPath)
if err == nil && !opts.NoHeadless {
	shouldUseHeadless := opts.Headless || IsSPA(string(htmlContent))
	if shouldUseHeadless {
		if IsHeadlessAvailable() {
			fmt.Printf("SPA detected, using headless mode for: %s\n", pageURL)
			if err := HeadlessRender(ctx, pageURL, htmlPath, opts.WaitFor, opts.WaitTimeout); err != nil {
				fmt.Printf("warning: headless render failed, using static HTML: %v\n", err)
			}
		} else if opts.Headless {
			fmt.Println("warning: --headless requested but Chrome not available")
		}
	}
}
```

This requires passing options into `Crawl`. Update `Crawl` signature:

```go
type CrawlOptions struct {
	MaxDepth    int
	Headless    bool
	NoHeadless  bool
	WaitFor     string
	WaitTimeout int
}

func Crawl(ctx context.Context, site string, projectPath string, cookieJar *cookiejar.Jar, proxyString string, userAgent string, opts CrawlOptions) error {
```

Update `cmd/clone.go` to construct and pass `CrawlOptions`.

**Step 3: Run all tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./... -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add pkg/crawler/crawler.go cmd/clone.go cmd/root.go
git commit -m "feat: integrate SPA auto-detection with headless fallback

Auto-detect SPA pages and switch to chromedp rendering.
Add --headless, --no-headless, --wait-for, --wait-timeout flags.
Graceful degradation when Chrome is not available."
```

---

## Task 12: Filename Collision Handling

**Files:**
- Modify: `pkg/crawler/extractor.go`
- Create: `pkg/crawler/collision_test.go`

**Step 1: Write failing test**

Create `pkg/crawler/collision_test.go`:

```go
package crawler

import "testing"

func TestDeduplicateFilename(t *testing.T) {
	tracker := NewFilenameTracker()

	// First use — no change
	name1 := tracker.UniqueFilename("style.css", "https://example.com/v1/style.css")
	if name1 != "style.css" {
		t.Fatalf("expected style.css, got %s", name1)
	}

	// Same URL — same filename
	name2 := tracker.UniqueFilename("style.css", "https://example.com/v1/style.css")
	if name2 != "style.css" {
		t.Fatalf("expected same filename for same URL, got %s", name2)
	}

	// Different URL, same filename — should get hash suffix
	name3 := tracker.UniqueFilename("style.css", "https://example.com/v2/style.css")
	if name3 == "style.css" {
		t.Fatal("expected different filename for different URL")
	}
	if name3 == name1 {
		t.Fatal("collision not resolved")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestDeduplicateFilename -v`
Expected: FAIL — `NewFilenameTracker` not defined

**Step 3: Implement filename tracker in `extractor.go`**

Add to `pkg/crawler/extractor.go`:

```go
import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
)

// FilenameTracker deduplicates filenames by appending a short hash when collisions occur.
type FilenameTracker struct {
	mu    sync.Mutex
	names map[string]string // filename → first URL that claimed it
}

func NewFilenameTracker() *FilenameTracker {
	return &FilenameTracker{names: make(map[string]string)}
}

// UniqueFilename returns a unique filename. If the filename is already taken by a
// different URL, it appends a short hash of the URL before the extension.
func (ft *FilenameTracker) UniqueFilename(filename, sourceURL string) string {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	existing, ok := ft.names[filename]
	if !ok {
		ft.names[filename] = sourceURL
		return filename
	}
	if existing == sourceURL {
		return filename
	}

	// Collision — append hash
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(sourceURL)))[:6]
	unique := base + "-" + hash + ext
	ft.names[unique] = sourceURL
	return unique
}
```

**Step 4: Run test**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./pkg/crawler/ -run TestDeduplicateFilename -v`
Expected: PASS

**Step 5: Wire FilenameTracker into Extractor**

The `FilenameTracker` should be created once per crawl in `Crawl()` and passed to `Extractor` calls. Modify `Extractor` to accept it:

```go
func Extractor(link string, projectPath string, ft *FilenameTracker) error {
	// ... existing code ...
	base := parser.URLFilename(link)
	if ft != nil {
		base = ft.UniqueFilename(base, link)
	}
	// ... rest of function ...
}
```

Update all `Extractor` callers in `collector.go` to pass the tracker.

**Step 6: Run all tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./... -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add pkg/crawler/extractor.go pkg/crawler/collision_test.go pkg/crawler/collector.go
git commit -m "feat: handle filename collisions with hash-based dedup

FilenameTracker appends short SHA hash when two different URLs
produce the same filename (e.g. /v1/style.css and /v2/style.css)."
```

---

## Task 13: Add Concurrency Controls

**Files:**
- Modify: `pkg/crawler/collector.go`

**Step 1: Add rate limiting and parallelism bounds to collector**

In `setUpCollector`, add colly's built-in rate limiting:

```go
func setUpCollector(c *colly.Collector, ctx context.Context, cookieJar *cookiejar.Jar, proxyString, userAgent string) {
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       100 * time.Millisecond,
	})

	// ... rest of existing setup ...
}
```

Add `"time"` to imports.

**Step 2: Run all tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./... -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add pkg/crawler/collector.go
git commit -m "feat: add rate limiting and concurrency bounds

Limit to 5 parallel requests with 100ms delay between requests
to prevent overwhelming target servers."
```

---

## Task 14: Progress Reporting & Summary

**Files:**
- Create: `pkg/crawler/stats.go`
- Modify: `pkg/crawler/crawler.go`

**Step 1: Create stats tracker**

Create `pkg/crawler/stats.go`:

```go
package crawler

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Stats tracks crawl progress.
type Stats struct {
	PagesDownloaded atomic.Int64
	AssetsDownloaded atomic.Int64
	Errors          atomic.Int64
	mu              sync.Mutex
	failedURLs      []string
}

func NewStats() *Stats {
	return &Stats{}
}

func (s *Stats) PageDone() { s.PagesDownloaded.Add(1) }
func (s *Stats) AssetDone() { s.AssetsDownloaded.Add(1) }

func (s *Stats) RecordError(url string) {
	s.Errors.Add(1)
	s.mu.Lock()
	s.failedURLs = append(s.failedURLs, url)
	s.mu.Unlock()
}

func (s *Stats) PrintSummary() {
	fmt.Printf("\n--- Clone Summary ---\n")
	fmt.Printf("Pages:  %d\n", s.PagesDownloaded.Load())
	fmt.Printf("Assets: %d\n", s.AssetsDownloaded.Load())
	fmt.Printf("Errors: %d\n", s.Errors.Load())
	if len(s.failedURLs) > 0 {
		fmt.Println("Failed URLs:")
		for _, u := range s.failedURLs {
			fmt.Printf("  - %s\n", u)
		}
	}
}
```

**Step 2: Wire stats into Crawl and Extractor**

Add `stats.PageDone()` after each page download, `stats.AssetDone()` after each resource extraction, and `stats.RecordError()` on failures. Call `stats.PrintSummary()` at end of `Crawl()`.

**Step 3: Run all tests**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test ./... -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add pkg/crawler/stats.go pkg/crawler/crawler.go pkg/crawler/collector.go
git commit -m "feat: add progress tracking and clone summary

Print page/asset/error counts at end of crawl.
List failed URLs for debugging."
```

---

## Task 15: Final Integration Test

**Files:**
- Create: `integration_test.go` (at repo root)

**Step 1: Write end-to-end integration test**

Create `integration_test.go`:

```go
//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goclone-dev/goclone/cmd"
	"github.com/goclone-dev/goclone/pkg/crawler"
	"github.com/goclone-dev/goclone/pkg/file"
	"github.com/goclone-dev/goclone/pkg/html"
)

func TestFullClonePipeline(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<link rel="stylesheet" href="/style.css">
		<link rel="icon" href="/favicon.ico">
		<script src="/app.js"></script>
		<img src="/hero.png" srcset="/hero-2x.png 2x">
		<a href="/about">About</a>
		</html>`))
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<link rel="stylesheet" href="/style.css">
		<script src="/app.js"></script>
		<a href="/">Home</a>
		</html>`))
	})
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(`body { background: url("/bg.jpg"); }
@font-face { src: url("/font.woff2"); }`))
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`console.log("app")`))
	})
	mux.HandleFunc("/hero.png", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("png"))
	})
	mux.HandleFunc("/hero-2x.png", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("png2x"))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ico"))
	})
	mux.HandleFunc("/bg.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("jpg"))
	})
	mux.HandleFunc("/font.woff2", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("woff2"))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Clone
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	projectPath := file.CreateProject("testsite")
	err := crawler.Crawl(context.Background(), ts.URL, projectPath, nil, "", "", crawler.CrawlOptions{MaxDepth: 1})
	if err != nil {
		t.Fatalf("Crawl failed: %v", err)
	}
	err = html.LinkRestructure(projectPath)
	if err != nil {
		t.Fatalf("LinkRestructure failed: %v", err)
	}

	// Verify files exist
	checks := map[string]bool{
		"index.html":        true,
		"about/index.html":  true,
		"css/style.css":     true,
		"js/app.js":         true,
		"imgs/hero.png":     true,
		"imgs/hero-2x.png":  true,
		"imgs/favicon.ico":  true,
		"imgs/bg.jpg":       true,
		"fonts/font.woff2":  true,
	}

	for path, required := range checks {
		exists := file.Exists(filepath.Join(projectPath, path))
		if required && !exists {
			t.Errorf("expected %s to exist", path)
		}
	}

	// Verify HTML was rewritten
	indexContent := file.GetFileContent(filepath.Join(projectPath, "index.html"))
	if !strings.Contains(indexContent, "css/style.css") {
		t.Error("index.html should reference css/style.css")
	}

	// Verify CSS was rewritten
	cssContent := file.GetFileContent(filepath.Join(projectPath, "css", "style.css"))
	if !strings.Contains(cssContent, "../imgs/bg.jpg") {
		t.Errorf("CSS should reference ../imgs/bg.jpg, got: %s", cssContent)
	}
	if !strings.Contains(cssContent, "../fonts/font.woff2") {
		t.Errorf("CSS should reference ../fonts/font.woff2, got: %s", cssContent)
	}
}
```

**Step 2: Run integration test**

Run: `cd /Users/vibhavverma/claude-projects/goclone && go test -tags integration -v`
Expected: PASS

**Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: add end-to-end integration test

Verifies full pipeline: multi-page crawl, all resource types,
HTML link rewriting, CSS url() rewriting, srcset handling."
```

---

## Summary

| Task | What it does |
|------|-------------|
| 1 | Extend file types from 7 to 25+ |
| 2 | Fix TLS, panic, permissions security issues |
| 3 | Discover favicons, preload, srcset, media in HTML |
| 4 | BFS page queue with depth control and dedup |
| 5 | Multi-page crawling integration |
| 6 | Rewrite links in ALL HTML files with relative paths |
| 7 | CSS url() extraction and rewriting engine |
| 8 | Integrate CSS rewriting into pipeline |
| 9 | SPA detection heuristic |
| 10 | Headless browser rendering via chromedp |
| 11 | Wire SPA detection + headless into crawl pipeline |
| 12 | Filename collision handling |
| 13 | Rate limiting and concurrency bounds |
| 14 | Progress reporting and summary |
| 15 | End-to-end integration test |
