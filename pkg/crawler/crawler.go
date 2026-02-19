package crawler

import (
	"context"
	"fmt"
	"net/http/cookiejar"
	"time"
)

// CrawlOptions contains all options for the crawl process
type CrawlOptions struct {
	Depth       int
	Parallel    int
	Delay       int // milliseconds
	Headless    bool
	NoHeadless  bool
	WaitFor     string
	WaitTimeout time.Duration
	Proxy       string
	UserAgent   string
}

// Crawl asks the necessary crawlers for collecting links for building the web page.
// It chooses between static (Colly) and headless (chromedp) modes based on flags and SPA detection.
func Crawl(ctx context.Context, site string, projectPath string, cookieJar *cookiejar.Jar, opts CrawlOptions) error {
	stats := NewCrawlStats()
	defer stats.PrintSummary()

	useHeadless := opts.Headless

	// Auto-detect SPA unless explicitly set
	if !opts.Headless && !opts.NoHeadless {
		score, err := DetectSPA(site)
		if err == nil && score.IsSPA() {
			fmt.Printf("SPA detected (score=%d): %v\n", score.Score, score.Reasons)
			fmt.Println("Switching to headless mode for full rendering...")
			useHeadless = true
		}
	}

	if useHeadless {
		return crawlHeadless(ctx, site, projectPath, opts, stats)
	}

	if opts.Depth > 0 {
		return crawlMultiPage(ctx, site, projectPath, cookieJar, opts, stats)
	}

	// Single-page static mode
	stats.AddPage()
	return CollectorWithOpts(ctx, site, projectPath, cookieJar, opts)
}

// crawlHeadless uses a headless browser to render the page and capture all network assets
func crawlHeadless(ctx context.Context, site string, projectPath string, opts CrawlOptions, stats *CrawlStats) error {
	headlessOpts := HeadlessOptions{
		WaitFor:     opts.WaitFor,
		WaitTimeout: opts.WaitTimeout,
		UserAgent:   opts.UserAgent,
	}

	result, err := HeadlessRender(ctx, site, projectPath, headlessOpts)
	if err != nil {
		return fmt.Errorf("headless render failed: %w", err)
	}

	stats.AddPage()
	for range result.CapturedAssets {
		stats.AddAsset(0) // Size tracking from headless is per-asset
	}

	return nil
}

// crawlMultiPage uses BFS to crawl multiple pages within the same domain
func crawlMultiPage(ctx context.Context, startURL string, projectPath string, cookieJar *cookiejar.Jar, opts CrawlOptions, stats *CrawlStats) error {
	queue, err := NewPageQueue(startURL, opts.Depth)
	if err != nil {
		return fmt.Errorf("failed to create page queue: %w", err)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		entry := queue.Dequeue()
		if entry == nil {
			break
		}

		fmt.Printf("Crawling page [depth=%d]: %s\n", entry.Depth, entry.URL)
		stats.AddPage()

		discovered, err := CollectorWithLinksAndOpts(ctx, entry.URL, projectPath, cookieJar, opts)
		if err != nil {
			stats.AddWarning(fmt.Sprintf("failed to crawl %s: %v", entry.URL, err))
			fmt.Printf("warning: failed to crawl %s: %v\n", entry.URL, err)
			continue
		}

		for _, link := range discovered {
			queue.Enqueue(link, entry.Depth+1)
		}
	}

	return nil
}
