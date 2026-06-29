// Decode HTML entities in plain-text strings (e.g. article titles) that arrive
// from the backend with entities still encoded. The article body is decoded
// automatically by DOMPurify + dangerouslySetInnerHTML; titles and other fields
// rendered as plain React text nodes are not. This helper handles numeric entities
// (&#8217;, &#x2019;) and named entities (&amp;, &quot;, &lt;, &gt;, &nbsp;)
// by parsing the string with DOMParser as text/html and reading back its text
// content. The parsed document is inert — scripts and resource loads (e.g. an
// <img onerror>) never run — so even a hostile title containing markup decodes
// safely, and a stray </textarea>-style sequence can't truncate the result.
export function decodeEntities(input: string): string {
  if (!input) return '';
  const doc = new DOMParser().parseFromString(input, 'text/html');
  return doc.body.textContent ?? '';
}
