package crawler

import (
	"net/url"
	"strings"
	"sync"
)

// PageEntry represents a page to be crawled in the BFS queue
type PageEntry struct {
	URL   string
	Depth int
}

// PageQueue implements a BFS queue for multi-page crawling with deduplication
type PageQueue struct {
	mu       sync.Mutex
	queue    []PageEntry
	visited  map[string]bool
	maxDepth int
	domain   string // only crawl same-domain pages
}

// NewPageQueue creates a new BFS queue rooted at the given URL
func NewPageQueue(startURL string, maxDepth int) (*PageQueue, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}

	q := &PageQueue{
		queue:    []PageEntry{{URL: startURL, Depth: 0}},
		visited:  map[string]bool{normalizeURL(startURL): true},
		maxDepth: maxDepth,
		domain:   parsed.Hostname(),
	}
	return q, nil
}

// Enqueue adds a URL to the queue if it hasn't been visited and is within depth limit
func (q *PageQueue) Enqueue(rawURL string, depth int) bool {
	if q.maxDepth > 0 && depth > q.maxDepth {
		return false
	}

	norm := normalizeURL(rawURL)
	if norm == "" {
		return false
	}

	// Only crawl same-domain pages
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() != q.domain {
		return false
	}

	// Skip non-HTTP(S) schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// Skip common non-page extensions
	if isNonPageURL(parsed.Path) {
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.visited[norm] {
		return false
	}
	q.visited[norm] = true
	q.queue = append(q.queue, PageEntry{URL: rawURL, Depth: depth})
	return true
}

// Dequeue returns the next page to crawl, or nil if empty
func (q *PageQueue) Dequeue() *PageEntry {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return nil
	}
	entry := q.queue[0]
	q.queue = q.queue[1:]
	return &entry
}

// Len returns the number of items remaining in the queue
func (q *PageQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// VisitedCount returns how many unique URLs have been seen
func (q *PageQueue) VisitedCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.visited)
}

// normalizeURL strips fragments and trailing slashes to deduplicate URLs
func normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parsed.Fragment = ""
	// Normalize trailing slash
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

// isNonPageURL returns true for URLs that are obviously not HTML pages
func isNonPageURL(path string) bool {
	lower := strings.ToLower(path)
	nonPageExts := []string{
		".css", ".js", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp",
		".avif", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".otf",
		".pdf", ".zip", ".tar", ".gz", ".mp4", ".webm", ".mp3", ".ogg",
		".glb", ".gltf", ".obj", ".fbx", ".hdr", ".ktx2",
	}
	for _, ext := range nonPageExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// URLToLocalPath converts a URL to a local file path for saving HTML pages.
// e.g., https://example.com/about → about/index.html
//
//	https://example.com/ → index.html
func URLToLocalPath(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "index.html"
	}

	p := strings.TrimRight(parsed.Path, "/")
	if p == "" || p == "/" {
		return "index.html"
	}

	// Remove leading slash
	p = strings.TrimLeft(p, "/")

	// If the path already has an .html extension, use it as-is
	if strings.HasSuffix(strings.ToLower(p), ".html") || strings.HasSuffix(strings.ToLower(p), ".htm") {
		return p
	}

	// Otherwise treat it as a directory with index.html
	return p + "/index.html"
}
