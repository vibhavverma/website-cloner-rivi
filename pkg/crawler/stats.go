package crawler

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CrawlStats tracks progress during a crawl session
type CrawlStats struct {
	PagesDownloaded  atomic.Int64
	AssetsDownloaded atomic.Int64
	BytesDownloaded  atomic.Int64
	Warnings         []string
	mu               sync.Mutex
	startTime        time.Time
}

// NewCrawlStats creates a new stats tracker
func NewCrawlStats() *CrawlStats {
	return &CrawlStats{
		startTime: time.Now(),
	}
}

// AddPage records a page download
func (s *CrawlStats) AddPage() {
	s.PagesDownloaded.Add(1)
}

// AddAsset records an asset download with its size
func (s *CrawlStats) AddAsset(bytes int64) {
	s.AssetsDownloaded.Add(1)
	s.BytesDownloaded.Add(bytes)
}

// AddWarning records a non-fatal warning
func (s *CrawlStats) AddWarning(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Warnings = append(s.Warnings, msg)
}

// PrintSummary outputs the crawl statistics
func (s *CrawlStats) PrintSummary() {
	elapsed := time.Since(s.startTime)

	fmt.Println()
	fmt.Println("=== Crawl Summary ===")
	fmt.Printf("  Pages:    %d\n", s.PagesDownloaded.Load())
	fmt.Printf("  Assets:   %d\n", s.AssetsDownloaded.Load())
	fmt.Printf("  Size:     %s\n", formatBytes(s.BytesDownloaded.Load()))
	fmt.Printf("  Duration: %s\n", elapsed.Truncate(time.Millisecond))

	s.mu.Lock()
	warnings := s.Warnings
	s.mu.Unlock()

	if len(warnings) > 0 {
		fmt.Printf("  Warnings: %d\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("    - %s\n", w)
		}
	}
	fmt.Println("=====================")
}

// formatBytes converts bytes to human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
