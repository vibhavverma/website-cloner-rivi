package parser

import (
	"fmt"
	"net/url"

	strutil "github.com/torden/go-strutil"
)

// ValidateURL checks for a valid url
func ValidateURL(url string) bool {
	/*
		>>> https://google.com
		<<< true

		>>> google.com
		<<< false
	*/
	if !strutil.NewStringValidator().IsValidURL(url) {
		return false
	}

	return true
}

// ValidateDomain checks for a valid domain
func ValidateDomain(domain string) bool {
	/*
		>>> google.com
		<<< true

		>>> google
		<<< false
	*/
	if !strutil.NewStringValidator().IsValidDomain(domain) {
		return false
	}

	return true
}

// CreateURL will take in a valid domain and return the URL
func CreateURL(domain string) string {
	/*
		>>> google.com - Valid Domain
		<<< https://google.com - Returned URL
	*/

	// concate nate https:// and the valid domain and return the now valid url
	return "https://" + domain
}

// GetDomainSafe takes in a URL and returns the domain or an error
func GetDomainSafe(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %q: %w", rawURL, err)
	}
	hostname := u.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("no hostname found in URL %q", rawURL)
	}
	return hostname, nil
}

// GetDomain takes in a valid URL and returns the domain of the url.
// Deprecated: Use GetDomainSafe instead to avoid panics.
func GetDomain(validurl string) string {
	domain, err := GetDomainSafe(validurl)
	if err != nil {
		panic(err)
	}
	return domain
}
