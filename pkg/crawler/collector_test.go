package crawler

import (
	"testing"
)

func TestParseSrcset(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "image-480w.jpg 480w, image-800w.jpg 800w",
			expected: []string{"image-480w.jpg", "image-800w.jpg"},
		},
		{
			input:    "image.jpg",
			expected: []string{"image.jpg"},
		},
		{
			input:    "small.jpg 1x, large.jpg 2x",
			expected: []string{"small.jpg", "large.jpg"},
		},
		{
			input:    "",
			expected: nil,
		},
		{
			input:    "  img1.png 100w ,  img2.png 200w  ",
			expected: []string{"img1.png", "img2.png"},
		},
	}

	for _, tt := range tests {
		result := parseSrcset(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseSrcset(%q) returned %d URLs, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i, url := range result {
			if url != tt.expected[i] {
				t.Errorf("parseSrcset(%q)[%d] = %q, want %q", tt.input, i, url, tt.expected[i])
			}
		}
	}
}
