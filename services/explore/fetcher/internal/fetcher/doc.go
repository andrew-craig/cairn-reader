// Package fetcher provides RSS feed fetching functionality.
// It implements rate-limited feed polling, priority-based feed selection
// (never-fetched first, then oldest), and automatic feed disabling
// after consecutive failures.
package fetcher
