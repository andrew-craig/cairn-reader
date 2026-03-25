// Package processor provides email content processing functionality.
package processor

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// trackerDomains are image host domains used exclusively for tracking pixels.
var trackerDomains = []string{
	"open.substack.com",
	"list-manage.com",
	"mandrillapp.com",
	"mailchimp.com",
	"sendgrid.net",
	"mailgun.org",
	"constantcontact.com",
	"campaign-archive.com",
	"klaviyomail.com",
	"beehiiv.com",
	"convertkit.com",
}

// unsubscribeKeywords are text patterns that indicate unsubscribe/footer links.
var unsubscribeKeywords = []string{
	"unsubscribe",
	"manage preferences",
	"update your preferences",
	"manage your subscription",
	"email preferences",
}

// artifactKeywords are patterns for email client artifacts to remove.
var artifactKeywords = []string{
	"view in browser",
	"view this email in your browser",
	"view online",
	"click here to view",
}

// EmailCleaner performs email-specific pre-cleaning before sanitization.
type EmailCleaner struct{}

// NewEmailCleaner creates a new EmailCleaner.
func NewEmailCleaner() *EmailCleaner {
	return &EmailCleaner{}
}

// Clean strips tracking pixels, unsubscribe footers, and email client artifacts
// from the provided HTML. It returns the cleaned HTML.
func (c *EmailCleaner) Clean(htmlBody string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return "", err
	}

	var nodesToRemove []*html.Node
	walkForRemoval(doc, &nodesToRemove)
	for _, n := range nodesToRemove {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// walkForRemoval traverses the HTML tree and collects nodes that should be removed.
func walkForRemoval(n *html.Node, remove *[]*html.Node) {
	if shouldRemove(n) {
		*remove = append(*remove, n)
		return // skip children — whole subtree goes away
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkForRemoval(c, remove)
	}
}

// shouldRemove returns true if the node represents a tracking pixel, an
// unsubscribe element, an email client artifact, or hidden preheader text.
func shouldRemove(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	switch strings.ToLower(n.Data) {
	case "img":
		return isTrackingPixel(n)
	case "a":
		return isUnsubscribeLink(n)
	case "p", "div", "span", "td", "li", "table", "tr":
		return isArtifactBlock(n)
	}
	return false
}

// isTrackingPixel returns true for 1×1 images or images from known tracker domains.
func isTrackingPixel(n *html.Node) bool {
	var width, height, src string
	for _, a := range n.Attr {
		switch a.Key {
		case "width":
			width = a.Val
		case "height":
			height = a.Val
		case "src":
			src = a.Val
		}
	}

	// 1×1 pixel
	if (width == "1" || width == "0") && (height == "1" || height == "0") {
		return true
	}

	// Known tracker domain
	srcLower := strings.ToLower(src)
	for _, domain := range trackerDomains {
		if strings.Contains(srcLower, domain) {
			return true
		}
	}
	return false
}

// isUnsubscribeLink returns true when a link's visible text matches
// unsubscribe/preferences patterns.
func isUnsubscribeLink(n *html.Node) bool {
	text := strings.ToLower(extractText(n))
	for _, kw := range unsubscribeKeywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// isArtifactBlock returns true for block-level nodes whose text content matches
// email client artifact patterns (e.g. "View in browser") or is a hidden
// preheader (display:none / max-height:0 style).
func isArtifactBlock(n *html.Node) bool {
	// Check for hidden preheader: inline style with display:none or max-height:0
	for _, a := range n.Attr {
		if a.Key == "style" {
			style := strings.ToLower(a.Val)
			if strings.Contains(style, "display:none") ||
				strings.Contains(style, "display: none") ||
				strings.Contains(style, "max-height:0") ||
				strings.Contains(style, "max-height: 0") ||
				strings.Contains(style, "overflow:hidden") && strings.Contains(style, "max-height") {
				return true
			}
		}
	}

	text := strings.ToLower(strings.TrimSpace(extractText(n)))
	if text == "" {
		return false
	}

	for _, kw := range artifactKeywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	// Unsubscribe text in a block element
	for _, kw := range unsubscribeKeywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// extractText returns the concatenated text content of a node and its descendants.
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(extractText(c))
	}
	return buf.String()
}
