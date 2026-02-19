package crawler

import (
	"testing"
)

func TestAnalyzeSPAIndicators(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		expectSPA  bool
		minScore   int
	}{
		{
			name: "React SPA with empty root",
			html: `<html><head><script src="bundle.js"></script><script src="vendor.js"></script><script src="app.js"></script></head>
<body><div id="root"></div><noscript>You need to enable JavaScript to run this app.</noscript></body></html>`,
			expectSPA: true,
			minScore:  5,
		},
		{
			name: "Next.js app",
			html: `<html><head></head><body><div id="__next"></div>
<script id="__NEXT_DATA__" type="application/json">{}</script></body></html>`,
			expectSPA: true,
			minScore:  5,
		},
		{
			name:      "Static HTML page with content",
			html:      `<html><head><link rel="stylesheet" href="style.css"></head><body><h1>Welcome</h1><p>This is a normal static page with plenty of text content that would indicate this is not a single page application at all. It has paragraphs and headings and links like a normal website.</p><a href="/about">About</a></body></html>`,
			expectSPA: false,
			minScore:  0,
		},
		{
			name: "Vue app with empty #app",
			html: `<html><head></head><body><div id="app"></div>
<script src="chunk-vendors.js"></script><script src="app.js"></script><script src="main.js"></script></body></html>`,
			expectSPA: true,
			minScore:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := AnalyzeSPAIndicators([]byte(tt.html))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if score.IsSPA() != tt.expectSPA {
				t.Errorf("IsSPA() = %v, want %v (score=%d, reasons=%v)",
					score.IsSPA(), tt.expectSPA, score.Score, score.Reasons)
			}
			if score.Score < tt.minScore {
				t.Errorf("score = %d, want >= %d (reasons=%v)",
					score.Score, tt.minScore, score.Reasons)
			}
		})
	}
}
