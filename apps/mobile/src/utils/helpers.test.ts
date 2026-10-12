import { generateId, extractDomain, isValidUrl, estimateReadingTime } from './helpers';

describe('generateId', () => {
  it('returns a non-empty string', () => {
    const id = generateId();
    expect(typeof id).toBe('string');
    expect(id.length).toBeGreaterThan(0);
  });

  it('generates unique IDs', () => {
    const ids = new Set(Array.from({ length: 100 }, () => generateId()));
    expect(ids.size).toBe(100);
  });

  it('contains a timestamp prefix and random suffix separated by a dash', () => {
    const id = generateId();
    const parts = id.split('-');
    expect(parts.length).toBe(2);
    // First part should be a numeric timestamp
    expect(Number(parts[0])).not.toBeNaN();
  });
});

describe('extractDomain', () => {
  it('extracts domain from a full URL', () => {
    expect(extractDomain('https://example.com/path/to/page')).toBe('example.com');
  });

  it('strips www prefix', () => {
    expect(extractDomain('https://www.example.com/page')).toBe('example.com');
  });

  it('preserves subdomains other than www', () => {
    expect(extractDomain('https://blog.example.com/post')).toBe('blog.example.com');
  });

  it('handles URLs with ports', () => {
    expect(extractDomain('http://localhost:3000/api')).toBe('localhost');
  });

  it('returns the original string for invalid URLs', () => {
    expect(extractDomain('not-a-url')).toBe('not-a-url');
  });

  it('handles URLs with query params and fragments', () => {
    expect(extractDomain('https://example.com/page?q=1#section')).toBe('example.com');
  });
});

describe('isValidUrl', () => {
  it('returns true for valid http URLs', () => {
    expect(isValidUrl('http://example.com')).toBe(true);
  });

  it('returns true for valid https URLs', () => {
    expect(isValidUrl('https://example.com/path?q=1')).toBe(true);
  });

  it('returns false for plain strings', () => {
    expect(isValidUrl('not a url')).toBe(false);
  });

  it('returns false for empty string', () => {
    expect(isValidUrl('')).toBe(false);
  });

  it('returns false for strings without protocol', () => {
    expect(isValidUrl('example.com')).toBe(false);
  });

  it('returns true for non-http protocols', () => {
    expect(isValidUrl('ftp://files.example.com/doc')).toBe(true);
  });
});
