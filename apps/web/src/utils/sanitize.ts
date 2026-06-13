import DOMPurify from 'dompurify';

// Force every external link to open in a new tab with a hardened rel, so a
// sanitized article can't reach back into the opener (reverse-tabnabbing) or
// leak a referrer. Registered once at module load; DOMPurify hooks are global.
DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  // Skip in-page anchors (#footnote, TOC links): forcing target=_blank on them
  // would open a blank new tab instead of scrolling to the target element.
  const href = node.getAttribute('href');
  if (node.tagName === 'A' && href && !href.startsWith('#')) {
    node.setAttribute('target', '_blank');
    node.setAttribute('rel', 'noopener noreferrer');
  }
});

// Sanitize article HTML before it is injected via dangerouslySetInnerHTML.
// DOMPurify strips <script>, inline event handlers (on*), and javascript:
// URLs by default; the hook above hardens surviving anchors. Factored out as a
// pure function so the security behaviour can be unit-tested in isolation.
export function sanitizeArticleHtml(html: string): string {
  return DOMPurify.sanitize(html);
}
