// utils/utils.go
package utils

import (
	"fmt"
	"math/rand"
	"net/url"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString creates a random string of a given length.
func GenerateRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// checks if the URL is valid and returns a sanitized URL.
func SanitizeURL(inputURL string) (string, error) {
	parsedURL, err := url.ParseRequestURI(inputURL)
	if err != nil {
		return "", err
	}

	// Check the scheme of the URL
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	// Return the sanitized URL
	return parsedURL.String(), nil
}
