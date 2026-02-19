# Perfect Frontend Cloning — Design Document

**Date:** 2026-02-19
**Approach:** Dual-mode (enhanced Colly + chromedp with smart SPA detection)

## Problem

goclone currently clones a fraction of a website's frontend:
- Only 7 file extensions supported (missing fonts, webp, ico, etc.)
- Only `index.html` is cloned — no multi-page support
- CSS `url()` references (background images, fonts) are never rewritten
- `srcset` attributes (responsive images) are ignored
- Inline `<style>` blocks with `url()` references are untouched
- SPAs render as empty shells (no JS execution)

## Design

### 1. Enhanced Resource Discovery & Extraction

**HTML attributes to scan:**
- `link[rel='stylesheet']` — CSS
- `link[rel='icon']` — Favicon
- `link[rel='preload']` — Fonts, critical assets
- `link[rel='manifest']` — PWA manifests
- `script[src]` — JS
- `img[src]` — Images
- `img[srcset]` — Responsive images
- `source[src]`, `source[srcset]` — Video/picture sources
- `video[src]`, `audio[src]` — Media
- `object[data]` — Embedded objects

**CSS url() scanning:**
After downloading CSS files, parse for `url()` references, `@import` statements, and `@font-face src` declarations. Download referenced resources and rewrite paths.

**Extended extension map (25+ types):**
`.css .js .jpg .jpeg .gif .png .svg .webp .avif .ico .woff .woff2 .ttf .eot .otf .json .webmanifest .mp4 .webm .ogg .mp3 .map`

### 2. Multi-Page Crawling

- BFS crawl from starting URL
- Discover same-domain `<a href="...">` links
- Save pages preserving URL path structure:
  - `/about` → `about/index.html`
  - `/blog/post-1` → `blog/post-1/index.html`
- Depth limit via `--depth` flag (default: no limit)
- Track visited URLs to avoid duplicates
- Stay on same domain

**Directory structure:**
```
project/
  index.html
  about/index.html
  blog/index.html
  blog/post-1/index.html
  assets/
    css/
    js/
    imgs/
    fonts/
```

### 3. SPA Detection & Headless Browser Mode

**Score-based detection (threshold: 3+):**
- +2: Body has single child div (id="root"|"app"|"__next"|"__nuxt")
- +2: Body text content < 100 chars (empty shell)
- +1: Script tags with "bundle" or "chunk" in filename
- +1: `<noscript>` tag with "enable JavaScript" text
- +1: Meta tag with "react"|"vue"|"angular"|"next"|"nuxt"
- -2: Body has > 500 chars of visible text (server-rendered)

**Headless mode (chromedp):**
1. Launch headless Chrome
2. Navigate to URL
3. Wait for network idle (no requests for 2s)
4. Optional: wait for specific selector (`--wait-for` flag)
5. Extract fully rendered DOM
6. Capture all network requests → download resources
7. Save rendered HTML, feed into link-rewriting pipeline

**Fallback:** If chromedp/Chrome not available, warn and continue with static mode.

### 4. CSS Rewriting

**New `pkg/css/rewriter.go`:**
- Parse CSS for `url()` references
- Resolve to absolute URLs, download resources
- Compute relative path from CSS file to saved resource
- Rewrite `url()` to local path
- Handle `@import` chains recursively

**Enhanced HTML rewriting (`arrange.go`):**
- Process ALL `.html` files (not just index.html)
- Parse and rewrite `srcset` attributes
- Scan inline `<style>` blocks for `url()` references
- Scan inline `style=""` attributes for `url()` references
- Rewrite `<a href>` to local page paths

### 5. Filename Collision Handling

Maintain map of `filename → URL`. On collision, append short hash:
`style.css` → `style-a3f2.css`, `style-b7c1.css`

## Files

**New:**
- `pkg/css/rewriter.go` — CSS url() parsing & rewriting
- `pkg/crawler/headless.go` — chromedp SPA rendering
- `pkg/crawler/detector.go` — SPA detection heuristic
- `pkg/crawler/queue.go` — BFS page queue with depth tracking

**Modified:**
- `pkg/crawler/collector.go` — Extended HTML attribute scanning
- `pkg/crawler/extractor.go` — 25+ file extensions, collision handling
- `pkg/crawler/html.go` — Remove global TLS hack, use proper HTTP client
- `pkg/html/arrange.go` — Multi-file, srcset, inline style rewriting
- `pkg/file/write.go` — New directory structure
- `cmd/clone.go` — New flags
- `cmd/root.go` — Register new flags
- `go.mod` — Add chromedp dependency

## New CLI Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--depth N` | no limit | Max crawl depth |
| `--headless` | auto-detect | Force headless Chrome mode |
| `--no-headless` | false | Disable headless entirely |
| `--wait-for "sel"` | none | CSS selector to wait for in headless mode |
| `--wait-timeout` | 30s | Max render wait time |
