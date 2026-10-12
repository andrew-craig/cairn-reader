import { describe, it, expect } from 'vitest';
import { formatDate, formatPublishedDate, pluralize } from './date';

const ONE_DAY_MS = 24 * 60 * 60 * 1000;

describe('formatDate', () => {
  it('returns "Today" for current timestamp', () => {
    expect(formatDate(Date.now())).toBe('Today');
  });

  it('returns "Today" for a timestamp slightly in the future', () => {
    expect(formatDate(Date.now() + 2 * 60 * 60 * 1000)).toBe('Today');
  });

  it('returns "Yesterday" for timestamp one day ago', () => {
    expect(formatDate(Date.now() - ONE_DAY_MS)).toBe('Yesterday');
  });

  it('returns "X days ago" for 2-6 days ago', () => {
    expect(formatDate(Date.now() - 3 * ONE_DAY_MS)).toBe('3 days ago');
  });

  it('returns "1 week ago" for 7-13 days ago', () => {
    expect(formatDate(Date.now() - 7 * ONE_DAY_MS)).toBe('1 week ago');
  });

  it('returns "X weeks ago" for 14-29 days ago', () => {
    expect(formatDate(Date.now() - 14 * ONE_DAY_MS)).toBe('2 weeks ago');
  });

  it('returns "1 month ago" for 30-59 days ago', () => {
    expect(formatDate(Date.now() - 30 * ONE_DAY_MS)).toBe('1 month ago');
  });

  it('returns "X months ago" for 60-364 days ago', () => {
    expect(formatDate(Date.now() - 90 * ONE_DAY_MS)).toBe('3 months ago');
  });

  it('returns "1 year ago" for 365-729 days ago', () => {
    expect(formatDate(Date.now() - 365 * ONE_DAY_MS)).toBe('1 year ago');
  });

  it('returns "X years ago" for 730+ days ago', () => {
    expect(formatDate(Date.now() - 730 * ONE_DAY_MS)).toBe('2 years ago');
  });
});

describe('formatPublishedDate', () => {
  it('formats an ISO date relative to now', () => {
    expect(formatPublishedDate(new Date(Date.now() - 10 * ONE_DAY_MS).toISOString())).toBe(
      '1 week ago'
    );
  });

  it('returns null when the date is missing', () => {
    expect(formatPublishedDate(undefined)).toBeNull();
    expect(formatPublishedDate('')).toBeNull();
  });

  it('returns null when the date is unparseable', () => {
    expect(formatPublishedDate('not a date')).toBeNull();
  });
});

describe('pluralize', () => {
  it('pluralizes zero and many', () => {
    expect(pluralize(0, 'subscription')).toBe('subscriptions');
    expect(pluralize(2, 'subscription')).toBe('subscriptions');
  });

  it('keeps the singular for one', () => {
    expect(pluralize(1, 'subscription')).toBe('subscription');
  });

  it('uses a custom plural', () => {
    expect(pluralize(2, 'entry', 'entries')).toBe('entries');
    expect(pluralize(1, 'entry', 'entries')).toBe('entry');
  });
});
