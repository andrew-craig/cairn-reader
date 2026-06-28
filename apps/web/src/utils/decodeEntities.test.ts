import { describe, expect, it } from 'vitest';
import { decodeEntities } from './decodeEntities';

describe('decodeEntities', () => {
  it('decodes numeric decimal entity &#8217; to curly apostrophe', () => {
    expect(decodeEntities('&#8217;')).toBe('’');
  });

  it('decodes numeric hex entity &#x2019; to curly apostrophe', () => {
    expect(decodeEntities('&#x2019;')).toBe('’');
  });

  it('decodes &amp; to &', () => {
    expect(decodeEntities('&amp;')).toBe('&');
  });

  it('decodes &quot; to "', () => {
    expect(decodeEntities('&quot;')).toBe('"');
  });

  it('passes plain text through unchanged', () => {
    expect(decodeEntities('Hello World')).toBe('Hello World');
  });

  it('returns empty string for empty input', () => {
    expect(decodeEntities('')).toBe('');
  });

  it('decodes a realistic article title', () => {
    expect(decodeEntities('Charlie Kirk&#8217;s legacy')).toBe("Charlie Kirk’s legacy");
  });
});
