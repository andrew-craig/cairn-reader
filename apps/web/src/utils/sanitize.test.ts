import { describe, expect, it } from 'vitest';
import { sanitizeArticleHtml } from './sanitize';

describe('sanitizeArticleHtml', () => {
  it('strips <script> tags', () => {
    const dirty = '<p>Hello</p><script>window.stolen = document.cookie;</script>';
    const clean = sanitizeArticleHtml(dirty);
    expect(clean).toContain('<p>Hello</p>');
    expect(clean).not.toContain('<script');
    expect(clean).not.toContain('document.cookie');
  });

  it('strips inline event handlers (onerror)', () => {
    const dirty = '<img src="x" onerror="alert(1)" />';
    const clean = sanitizeArticleHtml(dirty);
    expect(clean).not.toContain('onerror');
    expect(clean).not.toContain('alert(1)');
  });

  it('strips javascript: URLs', () => {
    const dirty = '<a href="javascript:alert(1)">click</a>';
    const clean = sanitizeArticleHtml(dirty);
    expect(clean).not.toContain('javascript:');
  });

  it('keeps safe content and hardens external links', () => {
    const clean = sanitizeArticleHtml('<a href="https://example.com">link</a>');
    expect(clean).toContain('href="https://example.com"');
    expect(clean).toContain('target="_blank"');
    expect(clean).toContain('rel="noopener noreferrer"');
  });

  it('leaves in-page anchor links untouched', () => {
    const clean = sanitizeArticleHtml('<a href="#footnote-1">jump</a>');
    expect(clean).toContain('href="#footnote-1"');
    expect(clean).not.toContain('target="_blank"');
  });
});
