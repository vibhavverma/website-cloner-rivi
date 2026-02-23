# Website Cloner

## Structure
- `cmd/goclone/` — CLI entry point (cobra commands)
- `pkg/` — Core packages: crawler, css, file, html, parser, server
- `testutils/` — Test helpers

## Build & Test
- Build: `go build -o goclone ./cmd/goclone`
- Test: `go test ./... -v`
- Module: `github.com/goclone-dev/goclone` (Go 1.24)

## Key Dependencies
- chromedp — headless browser rendering
- colly — web crawling
- cobra — CLI framework
- goquery — HTML parsing

## Deploy
- Binary distribution via `go install`
- GoReleaser config in `.goreleaser.yml`
