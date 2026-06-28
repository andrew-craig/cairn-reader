// Decode HTML entities in plain-text strings (e.g. article titles) that arrive
// from the backend with entities still encoded. The article body is decoded
// automatically by DOMPurify + dangerouslySetInnerHTML; titles and other fields
// rendered as plain React text nodes are not. This helper handles numeric entities
// (&#8217;, &#x2019;) and named entities (&amp;, &quot;, &lt;, &gt;, &nbsp;)
// by running the string through a detached textarea, which lets the browser's
// built-in HTML parser do the work without touching the DOM tree.
export function decodeEntities(input: string): string {
  if (!input) return '';
  const textarea = document.createElement('textarea');
  textarea.innerHTML = input;
  return textarea.value;
}
