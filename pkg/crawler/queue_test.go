package crawler

import (
	"testing"
)

func TestNewPageQueue(t *testing.T) {
	q, err := NewPageQueue("https://example.com", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Len() != 1 {
		t.Errorf("expected queue length 1, got %d", q.Len())
	}
	if q.VisitedCount() != 1 {
		t.Errorf("expected 1 visited, got %d", q.VisitedCount())
	}
}

func TestEnqueueDedup(t *testing.T) {
	q, _ := NewPageQueue("https://example.com", 5)

	// Same URL with fragment should be deduped
	if !q.Enqueue("https://example.com/about", 1) {
		t.Error("expected /about to be enqueued")
	}
	if q.Enqueue("https://example.com/about#section", 1) {
		t.Error("expected /about#section to be deduped")
	}
	// Trailing slash should be deduped
	if q.Enqueue("https://example.com/about/", 1) {
		t.Error("expected /about/ to be deduped with /about")
	}
}

func TestEnqueueDepthLimit(t *testing.T) {
	q, _ := NewPageQueue("https://example.com", 2)

	if !q.Enqueue("https://example.com/a", 1) {
		t.Error("depth 1 should be within limit")
	}
	if !q.Enqueue("https://example.com/b", 2) {
		t.Error("depth 2 should be within limit")
	}
	if q.Enqueue("https://example.com/c", 3) {
		t.Error("depth 3 should exceed limit")
	}
}

func TestEnqueueCrossDomain(t *testing.T) {
	q, _ := NewPageQueue("https://example.com", 5)

	if q.Enqueue("https://other.com/page", 1) {
		t.Error("cross-domain URL should be rejected")
	}
	if !q.Enqueue("https://example.com/page", 1) {
		t.Error("same-domain URL should be accepted")
	}
}

func TestEnqueueSkipsAssets(t *testing.T) {
	q, _ := NewPageQueue("https://example.com", 5)

	assetURLs := []string{
		"https://example.com/style.css",
		"https://example.com/app.js",
		"https://example.com/photo.jpg",
		"https://example.com/model.glb",
	}
	for _, u := range assetURLs {
		if q.Enqueue(u, 1) {
			t.Errorf("expected asset URL %q to be rejected", u)
		}
	}
}

func TestDequeue(t *testing.T) {
	q, _ := NewPageQueue("https://example.com", 5)
	q.Enqueue("https://example.com/about", 1)
	q.Enqueue("https://example.com/contact", 1)

	entry := q.Dequeue()
	if entry == nil || entry.URL != "https://example.com" {
		t.Errorf("expected first entry to be root URL, got %v", entry)
	}

	entry = q.Dequeue()
	if entry == nil || entry.URL != "https://example.com/about" {
		t.Errorf("expected second entry to be /about, got %v", entry)
	}

	entry = q.Dequeue()
	if entry == nil || entry.URL != "https://example.com/contact" {
		t.Errorf("expected third entry to be /contact, got %v", entry)
	}

	entry = q.Dequeue()
	if entry != nil {
		t.Errorf("expected nil on empty queue, got %v", entry)
	}
}

func TestURLToLocalPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "index.html"},
		{"https://example.com/", "index.html"},
		{"https://example.com/about", "about/index.html"},
		{"https://example.com/about/", "about/index.html"},
		{"https://example.com/blog/post-1", "blog/post-1/index.html"},
		{"https://example.com/page.html", "page.html"},
		{"https://example.com/docs/intro.htm", "docs/intro.htm"},
	}

	for _, tt := range tests {
		got := URLToLocalPath(tt.input)
		if got != tt.expected {
			t.Errorf("URLToLocalPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/about#section", "https://example.com/about"},
		{"https://example.com/about/", "https://example.com/about"},
		{"https://example.com/", "https://example.com/"},
		{"https://example.com", "https://example.com/"},
	}

	for _, tt := range tests {
		got := normalizeURL(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
