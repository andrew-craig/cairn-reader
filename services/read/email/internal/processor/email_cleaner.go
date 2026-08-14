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

// EmailCleaner performs email-specific pre-cleaning before sanitization.
type EmailCleaner struct{}

// NewEmailCleaner creates a new EmailCleaner.
func NewEmailCleaner() *EmailCleaner {
	return &EmailCleaner{}
}

// Clean strips tracking pixels and hidden preheader text from the provided
// HTML. It returns the cleaned HTML.
func (c *EmailCleaner) Clean(htmlBody string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return "", err
	}

	var nodesToRemove []*html.Node
	walkForRemoval(doc, &nodesToRemove, 0)
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
// depth is bounded by maxHTMLWalkDepth to guard against attacker-controlled nesting.
func walkForRemoval(n *html.Node, remove *[]*html.Node, depth int) {
	if depth > maxHTMLWalkDepth {
		return
	}
	if shouldRemove(n) {
		*remove = append(*remove, n)
		return // skip children — whole subtree goes away
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkForRemoval(c, remove, depth+1)
	}
}

// shouldRemove returns true if the node represents a tracking pixel or hidden
// preheader text.
func shouldRemove(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	switch strings.ToLower(n.Data) {
	case "img":
		return isTrackingPixel(n)
	case "p", "div", "span", "td", "li", "table", "tr":
		// Hidden preheader text is junk regardless of size — remove the subtree.
		return isHiddenPreheader(n)
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

// isHiddenPreheader returns true for block-level nodes hidden via inline style
// (display:none / max-height:0), which hold inbox preview text.
func isHiddenPreheader(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "style" {
			style := strings.ToLower(a.Val)
			if strings.Contains(style, "display:none") ||
				strings.Contains(style, "display: none") ||
				strings.Contains(style, "max-height:0") ||
				strings.Contains(style, "max-height: 0") {
				return true
			}
		}
	}
	return false
}
