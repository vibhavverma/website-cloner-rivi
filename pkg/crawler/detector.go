package crawler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// SPAScore represents the result of SPA detection
type SPAScore struct {
	Score   int      // Higher = more likely SPA
	Reasons []string // Human-readable reasons
}

// IsSPA returns true if the score suggests this is a single-page application
func (s SPAScore) IsSPA() bool {
	return s.Score >= 3
}

// DetectSPA fetches the URL and analyzes the HTML for SPA indicators
func DetectSPA(url string) (SPAScore, error) {
	resp, err := http.Get(url)
	if err != nil {
		return SPAScore{}, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SPAScore{}, fmt.Errorf("failed to read body: %w", err)
	}

	return AnalyzeSPAIndicators(body)
}

// AnalyzeSPAIndicators checks HTML content for SPA framework indicators
func AnalyzeSPAIndicators(htmlContent []byte) (SPAScore, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlContent))
	if err != nil {
		return SPAScore{}, err
	}

	var score SPAScore
	htmlStr := string(htmlContent)

	// Check for framework-specific root elements
	spaRootIDs := []string{"root", "app", "__next", "__nuxt", "svelte", "ember-application"}
	doc.Find("div[id]").Each(func(i int, s *goquery.Selection) {
		id, _ := s.Attr("id")
		for _, spaID := range spaRootIDs {
			if id == spaID {
				// Check if the root div is empty or nearly empty
				innerHTML, _ := s.Html()
				trimmed := strings.TrimSpace(innerHTML)
				if len(trimmed) < 50 {
					score.Score += 3
					score.Reasons = append(score.Reasons, fmt.Sprintf("empty SPA root element #%s", id))
				} else {
					score.Score += 1
					score.Reasons = append(score.Reasons, fmt.Sprintf("SPA root element #%s (with content)", id))
				}
				break
			}
		}
	})

	// Check body text content — SPAs have very little static text
	bodyText := doc.Find("body").Text()
	trimmedBody := strings.TrimSpace(bodyText)
	if len(trimmedBody) < 100 {
		score.Score += 2
		score.Reasons = append(score.Reasons, "very little text content in body")
	}

	// Count script tags — SPAs tend to have many/large scripts
	scriptCount := doc.Find("script[src]").Length()
	if scriptCount >= 3 {
		score.Score += 1
		score.Reasons = append(score.Reasons, fmt.Sprintf("%d external scripts", scriptCount))
	}

	// Check for common SPA framework markers in scripts/meta
	frameworkMarkers := map[string]string{
		"__NEXT_DATA__":     "Next.js",
		"__NUXT__":          "Nuxt.js",
		"window.__INITIAL_STATE__": "Vue/Vuex",
		"react":             "React",
		"ng-app":            "Angular",
		"data-reactroot":    "React",
	}
	for marker, framework := range frameworkMarkers {
		if strings.Contains(htmlStr, marker) {
			score.Score += 2
			score.Reasons = append(score.Reasons, fmt.Sprintf("%s framework detected (%s)", framework, marker))
		}
	}

	// Check for noscript tags with content (SPA fallback message)
	doc.Find("noscript").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(strings.ToLower(text), "javascript") || strings.Contains(strings.ToLower(text), "enable") {
			score.Score += 2
			score.Reasons = append(score.Reasons, "noscript tag warns about JavaScript requirement")
		}
	})

	return score, nil
}
