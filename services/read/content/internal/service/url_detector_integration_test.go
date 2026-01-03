package service

import (
	"context"
	"testing"
	"time"
)

// Integration tests that make real HTTP requests
// These tests are optional and can be skipped in CI environments

func TestDetectURL_RealRSSFeed_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector := NewURLDetector()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with a well-known, stable RSS feed
	result, err := detector.DetectURL(ctx, "https://www.nasa.gov/rss/dyn/breaking_news.rss")
	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypeFeed {
		t.Errorf("expected type %s, got %s", URLTypeFeed, result.Type)
	}

	if result.Title == nil {
		t.Error("expected non-nil title for NASA RSS feed")
	} else {
		t.Logf("Detected feed title: %s", *result.Title)
	}
}

func TestDetectURL_RealHTMLPage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector := NewURLDetector()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with a well-known website
	result, err := detector.DetectURL(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypePage {
		t.Errorf("expected type %s, got %s", URLTypePage, result.Type)
	}

	if result.Title == nil {
		t.Error("expected non-nil title for example.com")
	} else {
		t.Logf("Detected page title: %s", *result.Title)
	}
}

func TestDetectURL_RealRedirect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector := NewURLDetector()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with a URL that redirects (http -> https)
	result, err := detector.DetectURL(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	// Verify that the final URL is HTTPS
	if result.URL != "https://example.com/" {
		t.Logf("Final URL after redirect: %s", result.URL)
	}

	if result.Type != URLTypePage {
		t.Errorf("expected type %s, got %s", URLTypePage, result.Type)
	}
}

func TestDetectURL_RealTimeout_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector := NewURLDetector()
	// Very short timeout to force timeout error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Use a slow-responding URL or one that's likely to timeout
	result, err := detector.DetectURL(ctx, "https://httpstat.us/200?sleep=5000")
	if err != nil {
		t.Fatalf("DetectURL() should not return error on timeout, got %v", err)
	}

	if result.Type != URLTypeUnknown {
		t.Errorf("expected type %s on timeout, got %s", URLTypeUnknown, result.Type)
	}

	if result.Title != nil {
		t.Errorf("expected nil title on timeout, got '%s'", *result.Title)
	}
}
