// Package processor provides email content processing functionality.
package processor

// EmailCleaner performs email-specific pre-cleaning.
// Responsibilities:
// - Strip tracking pixels (1x1 images, known tracker domains)
// - Remove unsubscribe links/footers
// - Remove email client artifacts (View in browser, etc.)
// - Strip inline styles specific to email rendering
