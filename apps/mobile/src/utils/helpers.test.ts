import { generateId, formatDate, extractDomain, isValidUrl, estimateReadingTime, pluralize } from './helpers';

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

describe('formatDate', () => {
  const ONE_DAY_MS = 24 * 60 * 60 * 1000;

  it('returns "Today" for current timestamp', () => {
    const now = Date.now();
    expect(formatDate(now)).toBe('Today');
  });

  it('returns "Yesterday" for timestamp one day ago', () => {
    const yesterday = Date.now() - ONE_DAY_MS;
    expect(formatDate(yesterday)).toBe('Yesterday');
  });

  it('returns "X days ago" for 2-6 days ago', () => {
    const threeDaysAgo = Date.now() - 3 * ONE_DAY_MS;
    expect(formatDate(threeDaysAgo)).toBe('3 days ago');
  });

  it('returns "1 week ago" for 7-13 days ago', () => {
    const sevenDaysAgo = Date.now() - 7 * ONE_DAY_MS;
    expect(formatDate(sevenDaysAgo)).toBe('1 week ago');
  });

  it('returns "X weeks ago" for 14-29 days ago', () => {
    const twoWeeksAgo = Date.now() - 14 * ONE_DAY_MS;
    expect(formatDate(twoWeeksAgo)).toBe('2 weeks ago');
  });

  it('returns "1 month ago" for 30-59 days ago', () => {
    const thirtyDaysAgo = Date.now() - 30 * ONE_DAY_MS;
    expect(formatDate(thirtyDaysAgo)).toBe('1 month ago');
  });

  it('returns "X months ago" for 60-364 days ago', () => {
    const ninetyDaysAgo = Date.now() - 90 * ONE_DAY_MS;
    expect(formatDate(ninetyDaysAgo)).toBe('3 months ago');
  });

  it('returns "1 year ago" for 365-729 days ago', () => {
    const oneYearAgo = Date.now() - 365 * ONE_DAY_MS;
    expect(formatDate(oneYearAgo)).toBe('1 year ago');
  });

  it('returns "X years ago" for 730+ days ago', () => {
    const twoYearsAgo = Date.now() - 730 * ONE_DAY_MS;
    expect(formatDate(twoYearsAgo)).toBe('2 years ago');
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

describe('pluralize', () => {
  it('returns the plural form for 0', () => {
    expect(pluralize(0, 'subscription')).toBe('subscriptions');
  });

  it('returns the singular form for 1', () => {
    expect(pluralize(1, 'subscription')).toBe('subscription');
  });

  it('returns the plural form for 2', () => {
    expect(pluralize(2, 'subscription')).toBe('subscriptions');
  });

  it('returns the plural form for large counts', () => {
    expect(pluralize(42, 'vote')).toBe('votes');
  });

  it('uses a custom plural form when provided', () => {
    expect(pluralize(2, 'entry', 'entries')).toBe('entries');
  });

  it('uses the singular form with a custom plural when count is 1', () => {
    expect(pluralize(1, 'entry', 'entries')).toBe('entry');
  });
});

describe('estimateReadingTime', () => {
  it('returns 1 minute for short text', () => {
    const shortText = 'Hello world';
    expect(estimateReadingTime(shortText)).toBe(1);
  });

  it('returns correct time for 200 words (1 minute)', () => {
    const words = Array(200).fill('word').join(' ');
    expect(estimateReadingTime(words)).toBe(1);
  });

  it('returns correct time for 400 words (2 minutes)', () => {
    const words = Array(400).fill('word').join(' ');
    expect(estimateReadingTime(words)).toBe(2);
  });

  it('rounds up partial minutes', () => {
    // 201 words / 200 wpm = 1.005 -> ceil -> 2
    const words = Array(201).fill('word').join(' ');
    expect(estimateReadingTime(words)).toBe(2);
  });

  it('handles text with varied whitespace', () => {
    const text = 'word1  word2\tword3\nword4';
    // split(/\s+/) yields 4 words -> ceil(4/200) = 1
    expect(estimateReadingTime(text)).toBe(1);
  });
});
