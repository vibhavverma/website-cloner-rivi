package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// extractResource is a helper to download a resource and log warnings on failure
func extractResource(kind, link string, absURL string, projectPath string) {
	fmt.Printf("%s found --> %s\n", kind, link)
	if err := Extractor(absURL, projectPath); err != nil {
		fmt.Printf("warning: failed to extract %s: %v\n", link, err)
	}
}

// parseSrcset parses an HTML srcset attribute and returns the list of URLs
func parseSrcset(srcset string) []string {
	var urls []string
	for _, candidate := range strings.Split(srcset, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		parts := strings.Fields(candidate)
		if len(parts) > 0 {
			urls = append(urls, parts[0])
		}
	}
	return urls
}

// registerResourceHandlers attaches OnHTML handlers for CSS, JS, images, icons,
// preloads, media, etc. to the given collector
func registerResourceHandlers(c *colly.Collector, projectPath string) {
	// CSS stylesheets
	c.OnHTML("link[rel='stylesheet']", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		extractResource("CSS", link, e.Request.AbsoluteURL(link), projectPath)
	})

	// JavaScript
	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		extractResource("JS", link, e.Request.AbsoluteURL(link), projectPath)
	})

	// Images (src)
	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		if strings.HasPrefix(link, "data:") || strings.HasPrefix(link, "blob:") {
			return
		}
		extractResource("IMG", link, e.Request.AbsoluteURL(link), projectPath)
	})

	// Images (srcset) — responsive images
	c.OnHTML("img[srcset]", func(e *colly.HTMLElement) {
		for _, src := range parseSrcset(e.Attr("srcset")) {
			if strings.HasPrefix(src, "data:") {
				continue
			}
			extractResource("IMG(srcset)", src, e.Request.AbsoluteURL(src), projectPath)
		}
	})

	// Picture source elements (srcset)
	c.OnHTML("source[srcset]", func(e *colly.HTMLElement) {
		for _, src := range parseSrcset(e.Attr("srcset")) {
			if strings.HasPrefix(src, "data:") {
				continue
			}
			extractResource("SOURCE(srcset)", src, e.Request.AbsoluteURL(src), projectPath)
		}
	})

	// Source elements with src (video/audio sources)
	c.OnHTML("source[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		extractResource("SOURCE", link, e.Request.AbsoluteURL(link), projectPath)
	})

	// Favicon and icons
	c.OnHTML("link[rel='icon'], link[rel='shortcut icon'], link[rel='apple-touch-icon']", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link != "" {
			extractResource("ICON", link, e.Request.AbsoluteURL(link), projectPath)
		}
	})

	// Preloaded resources (fonts, images, etc.)
	c.OnHTML("link[rel='preload']", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link != "" {
			extractResource("PRELOAD", link, e.Request.AbsoluteURL(link), projectPath)
		}
	})

	// Video with src attribute
	c.OnHTML("video[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		extractResource("VIDEO", link, e.Request.AbsoluteURL(link), projectPath)
	})

	// Video poster images
	c.OnHTML("video[poster]", func(e *colly.HTMLElement) {
		link := e.Attr("poster")
		if link != "" {
			extractResource("POSTER", link, e.Request.AbsoluteURL(link), projectPath)
		}
	})

	// Audio with src attribute
	c.OnHTML("audio[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		extractResource("AUDIO", link, e.Request.AbsoluteURL(link), projectPath)
	})
}

// Collector searches for css, js, images, and other resources within a given link
func Collector(ctx context.Context, url string, projectPath string, cookieJar *cookiejar.Jar, proxyString string, userAgent string) error {
	// First, download the main HTML file
	fmt.Printf("Downloading main HTML from: %s\n", url)
	if err := HTMLExtractor(url, projectPath); err != nil {
		return fmt.Errorf("failed to download main HTML: %v", err)
	}

	c := colly.NewCollector(colly.Async(true))
	setUpCollector(c, ctx, cookieJar, proxyString, userAgent)
	registerResourceHandlers(c, projectPath)

	if err := c.Visit(url); err != nil {
		return err
	}
	c.Wait()
	return nil
}

// CollectorWithOpts is like Collector but with concurrency controls
func CollectorWithOpts(ctx context.Context, url string, projectPath string, cookieJar *cookiejar.Jar, opts CrawlOptions) error {
	fmt.Printf("Downloading main HTML from: %s\n", url)
	if err := HTMLExtractor(url, projectPath); err != nil {
		return fmt.Errorf("failed to download main HTML: %v", err)
	}

	c := colly.NewCollector(colly.Async(true))
	setUpCollector(c, ctx, cookieJar, opts.Proxy, opts.UserAgent)
	setUpCollectorWithLimits(c, opts.Parallel, opts.Delay)
	registerResourceHandlers(c, projectPath)

	if err := c.Visit(url); err != nil {
		return err
	}
	c.Wait()
	return nil
}

// CollectorWithLinksAndOpts is like CollectorWithLinks but with concurrency controls
func CollectorWithLinksAndOpts(ctx context.Context, pageURL string, projectPath string, cookieJar *cookiejar.Jar, opts CrawlOptions) ([]string, error) {
	localPath := URLToLocalPath(pageURL)
	fmt.Printf("Downloading HTML from: %s → %s\n", pageURL, localPath)
	if err := HTMLExtractorToPath(pageURL, projectPath, localPath); err != nil {
		return nil, fmt.Errorf("failed to download HTML: %v", err)
	}

	c := colly.NewCollector(colly.Async(true))
	setUpCollector(c, ctx, cookieJar, opts.Proxy, opts.UserAgent)
	setUpCollectorWithLimits(c, opts.Parallel, opts.Delay)
	registerResourceHandlers(c, projectPath)

	var mu sync.Mutex
	var links []string

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "javascript:") || strings.HasPrefix(link, "mailto:") {
			return
		}
		absLink := e.Request.AbsoluteURL(link)
		if absLink != "" {
			mu.Lock()
			links = append(links, absLink)
			mu.Unlock()
		}
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, err
	}
	c.Wait()
	return links, nil
}

// CollectorWithLinks is like Collector but also discovers and returns same-domain <a> links
func CollectorWithLinks(ctx context.Context, pageURL string, projectPath string, cookieJar *cookiejar.Jar, proxyString string, userAgent string) ([]string, error) {
	// Determine the local path for this page
	localPath := URLToLocalPath(pageURL)
	fmt.Printf("Downloading HTML from: %s → %s\n", pageURL, localPath)
	if err := HTMLExtractorToPath(pageURL, projectPath, localPath); err != nil {
		return nil, fmt.Errorf("failed to download HTML: %v", err)
	}

	c := colly.NewCollector(colly.Async(true))
	setUpCollector(c, ctx, cookieJar, proxyString, userAgent)
	registerResourceHandlers(c, projectPath)

	// Collect discovered links
	var mu sync.Mutex
	var links []string

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "javascript:") || strings.HasPrefix(link, "mailto:") {
			return
		}
		absLink := e.Request.AbsoluteURL(link)
		if absLink != "" {
			mu.Lock()
			links = append(links, absLink)
			mu.Unlock()
		}
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, err
	}
	c.Wait()
	return links, nil
}

type cancelableTransport struct {
	ctx       context.Context
	transport http.RoundTripper
}

func (t cancelableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.ctx.Err(); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req.WithContext(t.ctx))
}

func setUpCollector(c *colly.Collector, ctx context.Context, cookieJar *cookiejar.Jar, proxyString, userAgent string) {
	if cookieJar != nil {
		c.SetCookieJar(cookieJar)
	}
	if proxyString != "" {
		c.SetProxy(proxyString)
	} else {
		c.WithTransport(cancelableTransport{ctx: ctx, transport: http.DefaultTransport})
	}
	if userAgent != "" {
		c.UserAgent = userAgent
	}
}

// setUpCollectorWithLimits configures rate limiting on the collector
func setUpCollectorWithLimits(c *colly.Collector, parallel int, delayMs int) {
	if parallel <= 0 {
		parallel = 5
	}
	if delayMs < 0 {
		delayMs = 0
	}
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: parallel,
		Delay:       time.Duration(delayMs) * time.Millisecond,
	})
}
