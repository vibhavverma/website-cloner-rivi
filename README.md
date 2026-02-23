# Website Cloner

A Go CLI tool for cloning websites locally with full headless browser rendering support. Built on top of [goclone](https://github.com/goclone-dev/goclone) with chromedp for JavaScript-rendered pages and colly for efficient crawling.

## Features

- **Headless browser rendering** — Captures JavaScript-rendered content via chromedp
- **Multi-page crawling** — Follows links and downloads entire site structures using colly
- **Asset preservation** — Downloads HTML, CSS, JS, images, and other static assets
- **Relative link rewriting** — Maintains working navigation in the cloned copy
- **Local preview server** — Serve cloned sites locally for review

## Installation

```bash
go install github.com/vibhavverma/website-cloner-rivi/cmd/goclone@latest
```

Or clone and build:

```bash
git clone https://github.com/vibhavverma/website-cloner-rivi.git
cd website-cloner-rivi
go build -o goclone ./cmd/goclone
```

## Usage

```bash
# Clone a website
goclone https://example.com

# Clone with headless browser rendering
goclone --browser https://example.com
```

## Tech Stack

- **Go 1.24** — Core language
- **chromedp** — Headless Chrome for JS-rendered pages
- **colly** — Fast web crawler
- **cobra** — CLI framework
- **goquery** — HTML parsing

## Development

```bash
# Run tests
go test ./... -v

# Build
go build -o goclone ./cmd/goclone
```

## License

MIT — see [LICENSE](LICENSE) for details.
