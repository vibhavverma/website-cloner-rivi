package crawler

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// HeadlessOptions configures headless browser behavior
type HeadlessOptions struct {
	WaitFor     string        // CSS selector to wait for before capturing
	WaitTimeout time.Duration // Maximum time to wait for page load
	UserAgent   string
}

// HeadlessResult contains the results of headless rendering
type HeadlessResult struct {
	HTML           string   // Fully rendered HTML
	CapturedAssets []string // List of captured asset local paths
}

// HeadlessRender navigates to a URL using a headless browser, captures the
// fully rendered DOM, and intercepts all network requests to save runtime-loaded
// assets (3D models, textures, shaders, fonts, etc.)
func HeadlessRender(ctx context.Context, pageURL string, projectPath string, opts HeadlessOptions) (*HeadlessResult, error) {
	if opts.WaitTimeout == 0 {
		opts.WaitTimeout = 30 * time.Second
	}

	// Create browser context
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-web-security", true),
		)...,
	)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Set up timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(browserCtx, opts.WaitTimeout)
	defer timeoutCancel()

	result := &HeadlessResult{}
	var mu sync.Mutex

	// Set up network interception to capture all loaded assets
	chromedp.ListenTarget(timeoutCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			// Continue the request (don't block it)
			go func() {
				_ = chromedp.Run(timeoutCtx, fetch.ContinueRequest(e.RequestID))
			}()

		case *network.EventResponseReceived:
			go func() {
				resp := e.Response
				if resp == nil {
					return
				}

				assetURL := resp.URL
				if !shouldCaptureAsset(assetURL) {
					return
				}

				// Get the response body
				var body []byte
				if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
					var err error
					body, err = network.GetResponseBody(e.RequestID).Do(ctx)
					return err
				})); err != nil {
					// Response body may not be available for all requests
					return
				}

				if len(body) == 0 {
					return
				}

				// Save the asset preserving URL path structure
				localPath := urlToLocalPath(assetURL, projectPath)
				if localPath == "" {
					return
				}

				if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
					fmt.Printf("warning: failed to create dir for %s: %v\n", localPath, err)
					return
				}

				if err := os.WriteFile(localPath, body, 0644); err != nil {
					fmt.Printf("warning: failed to write %s: %v\n", localPath, err)
					return
				}

				mu.Lock()
				result.CapturedAssets = append(result.CapturedAssets, localPath)
				mu.Unlock()

				fmt.Printf("CAPTURED --> %s\n", assetURL)
			}()
		}
	})

	// Enable network tracking and fetch interception
	actions := []chromedp.Action{
		network.Enable(),
		fetch.Enable(),
	}

	// Set user agent if provided
	if opts.UserAgent != "" {
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetUserAgentOverride(opts.UserAgent).Do(ctx)
		}))
	}

	// Navigate to the page
	actions = append(actions, chromedp.Navigate(pageURL))

	// Wait for page to be ready
	if opts.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(opts.WaitFor, chromedp.ByQuery))
	} else {
		// Wait for network idle (no requests for 500ms)
		actions = append(actions, chromedp.Sleep(2*time.Second))
	}

	// Capture the fully rendered HTML
	var html string
	actions = append(actions, chromedp.OuterHTML("html", &html))

	fmt.Printf("Headless rendering: %s\n", pageURL)
	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		return nil, fmt.Errorf("headless render failed: %w", err)
	}

	// Save the rendered HTML
	result.HTML = html
	indexPath := filepath.Join(projectPath, "index.html")
	if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
		return nil, fmt.Errorf("failed to write rendered HTML: %w", err)
	}

	// Give a moment for any remaining async asset captures
	time.Sleep(1 * time.Second)

	fmt.Printf("Headless capture complete: %d assets captured\n", len(result.CapturedAssets))
	return result, nil
}

// shouldCaptureAsset determines if a network request should be captured based on URL
func shouldCaptureAsset(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	// Skip data: and blob: URIs
	if strings.HasPrefix(rawURL, "data:") || strings.HasPrefix(rawURL, "blob:") {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	ext := strings.ToLower(filepath.Ext(parsed.Path))
	if ext == "" {
		return false
	}

	return IsAssetExtension(ext)
}

// urlToLocalPath converts a URL to a local file path, preserving the URL path structure.
// For same-domain URLs: https://example.com/assets/model.glb → projectPath/assets/model.glb
// For cross-domain URLs: https://cdn.example.com/textures/wood.jpg → projectPath/textures/wood.jpg
func urlToLocalPath(rawURL string, projectPath string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Path == "" {
		return ""
	}

	// Clean the path
	p := strings.TrimLeft(parsed.Path, "/")
	if p == "" {
		return ""
	}

	// Sanitize: prevent directory traversal
	p = filepath.Clean(p)
	if strings.Contains(p, "..") {
		return ""
	}

	return filepath.Join(projectPath, p)
}
