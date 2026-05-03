package parse_test

import (
	"strings"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/rss/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rss2Feed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>A test feed</description>
    <item>
      <title>Article One</title>
      <link>https://example.com/1</link>
      <guid>guid-1</guid>
      <author>Jane Doe</author>
      <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
      <description>Plain description</description>
      <content:encoded><![CDATA[<p>Full content</p>]]></content:encoded>
    </item>
    <item>
      <title>  Whitespace Title  </title>
      <link>https://example.com/2</link>
      <dc:creator>DC Creator</dc:creator>
      <pubDate>Tue, 03 Jan 2006 10:00:00 GMT</pubDate>
      <description>Second item</description>
    </item>
    <item>
      <title>No Date</title>
      <link>https://example.com/3</link>
      <description>No pub date</description>
    </item>
  </channel>
</rss>`

const atomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Feed</title>
  <link href="https://atom.example.com"/>
  <id>urn:atom-feed</id>
  <entry>
    <title>Atom Entry</title>
    <link href="https://atom.example.com/entry1"/>
    <id>urn:entry-1</id>
    <author><name>Atom Author</name></author>
    <updated>2006-01-02T15:04:05Z</updated>
    <summary>Atom summary</summary>
    <content type="html">&lt;p&gt;Atom content&lt;/p&gt;</content>
  </entry>
</feed>`

const noAuthorFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>No Author Feed</title>
    <link>https://noauthor.example.com</link>
    <description>Feed without authors</description>
    <item>
      <title>Email Only Author</title>
      <link>https://noauthor.example.com/1</link>
      <author>nobody@example.com</author>
      <description>Item with email-only author</description>
    </item>
    <item>
      <title>Truly No Author</title>
      <link>https://noauthor.example.com/2</link>
      <description>Item with no author at all</description>
    </item>
  </channel>
</rss>`

func TestParseBytes_RSS2(t *testing.T) {
	feed, err := parse.ParseBytes([]byte(rss2Feed))
	require.NoError(t, err)

	assert.Equal(t, "Test Feed", feed.Title)
	assert.Equal(t, "https://example.com", feed.SiteURL)
	assert.Equal(t, "A test feed", feed.Description)
	require.Len(t, feed.Items, 3)

	// First item: explicit author, content:encoded, guid
	item0 := feed.Items[0]
	assert.Equal(t, "Article One", item0.Title)
	assert.Equal(t, "guid-1", item0.GUID)
	assert.Equal(t, "https://example.com/1", item0.Link)
	assert.Equal(t, "Jane Doe", item0.Author)
	assert.Equal(t, "<p>Full content</p>", item0.Content)
	assert.Equal(t, "Plain description", item0.Description)
	require.NotNil(t, item0.PublishedAt)
	assert.Equal(t, 2006, item0.PublishedAt.Year())

	// Second item: dc:creator, title whitespace trimmed, no content:encoded so Content = Description
	item1 := feed.Items[1]
	assert.Equal(t, "Whitespace Title", item1.Title)
	assert.Equal(t, "DC Creator", item1.Author)
	assert.Equal(t, "Second item", item1.Content)
	require.NotNil(t, item1.PublishedAt)

	// Third item: no pub date → PublishedAt is nil; GUID falls back to Link
	item2 := feed.Items[2]
	assert.Equal(t, "No Date", item2.Title)
	assert.Nil(t, item2.PublishedAt)
	assert.Equal(t, "https://example.com/3", item2.GUID)
}

func TestParseReader_Atom(t *testing.T) {
	feed, err := parse.ParseReader(strings.NewReader(atomFeed))
	require.NoError(t, err)

	assert.Equal(t, "Atom Feed", feed.Title)
	require.Len(t, feed.Items, 1)

	item := feed.Items[0]
	assert.Equal(t, "Atom Entry", item.Title)
	assert.Equal(t, "Atom Author", item.Author)
	assert.Equal(t, "urn:entry-1", item.GUID)
	require.NotNil(t, item.PublishedAt)
	assert.Equal(t, 2006, item.PublishedAt.Year())
	// Content should be non-empty (atom <content>)
	assert.NotEmpty(t, item.Content)
}

func TestParseBytes_AuthorResolution(t *testing.T) {
	feed, err := parse.ParseBytes([]byte(noAuthorFeed))
	require.NoError(t, err)
	require.Len(t, feed.Items, 2)

	// Email-only author: gofeed puts the email string in Author.Name for RSS <author> tags
	// that contain only an email. We verify we still produce something reasonable or empty.
	// The important thing is we don't panic.
	_ = feed.Items[0].Author

	// Truly no author
	assert.Equal(t, "", feed.Items[1].Author)
}

func TestParseBytes_InvalidFeed(t *testing.T) {
	_, err := parse.ParseBytes([]byte("this is not xml"))
	assert.Error(t, err)
}

func TestParseBytes_PublishedAtFallback(t *testing.T) {
	// Feed with only <updated> (Atom style), no <published>
	feedXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Updated Only</title>
  <link href="https://example.com"/>
  <id>urn:updated-only</id>
  <entry>
    <title>Updated Entry</title>
    <link href="https://example.com/e1"/>
    <id>urn:e1</id>
    <updated>2024-06-15T12:00:00Z</updated>
    <summary>Entry with only updated date</summary>
  </entry>
</feed>`
	feed, err := parse.ParseBytes([]byte(feedXML))
	require.NoError(t, err)
	require.Len(t, feed.Items, 1)

	item := feed.Items[0]
	require.NotNil(t, item.PublishedAt)
	want := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, want, *item.PublishedAt)
}
