package crawler

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCrawlStats(t *testing.T) {
	s := NewCrawlStats()

	s.AddPage()
	s.AddPage()
	s.AddAsset(1024)
	s.AddAsset(2048)
	s.AddWarning("test warning")

	if s.PagesDownloaded.Load() != 2 {
		t.Errorf("expected 2 pages, got %d", s.PagesDownloaded.Load())
	}
	if s.AssetsDownloaded.Load() != 2 {
		t.Errorf("expected 2 assets, got %d", s.AssetsDownloaded.Load())
	}
	if s.BytesDownloaded.Load() != 3072 {
		t.Errorf("expected 3072 bytes, got %d", s.BytesDownloaded.Load())
	}
	if len(s.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(s.Warnings))
	}
}
