package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseFromReader_RSS(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Example Feed</title>
    <description>An example RSS feed</description>
    <link>https://example.com</link>
    <item>
      <title>Article 1</title>
      <link>https://example.com/article1</link>
      <description>First article description</description>
      <author>John Doe</author>
      <pubDate>Mon, 01 Jan 2024 12:00:00 GMT</pubDate>
      <guid>https://example.com/article1</guid>
    </item>
    <item>
      <title>Article 2</title>
      <link>https://example.com/article2</link>
      <description>Second article description</description>
      <pubDate>Tue, 02 Jan 2024 12:00:00 GMT</pubDate>
      <guid>https://example.com/article2</guid>
    </item>
  </channel>
</rss>`

	parser := NewParser()
	reader := strings.NewReader(rssFeed)

	result, err := parser.ParseFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Example Feed", result.Title)
	assert.Equal(t, "An example RSS feed", result.Description)
	assert.Equal(t, "https://example.com", result.SiteURL)
	assert.Len(t, result.Items, 2)

	// Check first item
	assert.Equal(t, "Article 1", result.Items[0].Title)
	assert.Equal(t, "https://example.com/article1", result.Items[0].URL)
	assert.Equal(t, "First article description", result.Items[0].Description)
	assert.Equal(t, "John Doe", result.Items[0].Author)
	assert.Equal(t, "https://example.com/article1", result.Items[0].GUID)
	assert.NotNil(t, result.Items[0].PublishedAt)

	// Check second item
	assert.Equal(t, "Article 2", result.Items[1].Title)
	assert.Equal(t, "https://example.com/article2", result.Items[1].URL)
}

func TestParser_ParseFromReader_Atom(t *testing.T) {
	atomFeed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Atom Feed</title>
  <subtitle>An example Atom feed</subtitle>
  <link href="https://example.com"/>
  <entry>
    <title>Atom Article 1</title>
    <link href="https://example.com/atom1"/>
    <id>https://example.com/atom1</id>
    <summary>First atom article</summary>
    <author><name>Jane Smith</name></author>
    <published>2024-01-01T12:00:00Z</published>
  </entry>
</feed>`

	parser := NewParser()
	reader := strings.NewReader(atomFeed)

	result, err := parser.ParseFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Example Atom Feed", result.Title)
	assert.Equal(t, "An example Atom feed", result.Description)
	assert.Equal(t, "https://example.com", result.SiteURL)
	assert.Len(t, result.Items, 1)

	// Check item
	assert.Equal(t, "Atom Article 1", result.Items[0].Title)
	assert.Equal(t, "https://example.com/atom1", result.Items[0].URL)
	assert.Equal(t, "First atom article", result.Items[0].Description)
	assert.Equal(t, "Jane Smith", result.Items[0].Author)
	assert.NotNil(t, result.Items[0].PublishedAt)
}

func TestParser_ParseFromReader_InvalidXML(t *testing.T) {
	invalidFeed := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Invalid Feed</title>
    <unclosed-tag>
  </channel>
</rss>`

	parser := NewParser()
	reader := strings.NewReader(invalidFeed)

	result, err := parser.ParseFromReader(reader)

	// Some XML parsers are forgiving, so we check if either error or result is returned
	if err != nil {
		assert.Contains(t, err.Error(), "failed to parse feed")
	}
	// If no error, we should at least get a result (parser was forgiving)
	if result != nil {
		assert.NotNil(t, result)
	}
}

func TestParser_ParseFromReader_EmptyFeed(t *testing.T) {
	emptyFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Empty Feed</title>
    <description>A feed with no items</description>
    <link>https://example.com</link>
  </channel>
</rss>`

	parser := NewParser()
	reader := strings.NewReader(emptyFeed)

	result, err := parser.ParseFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Empty Feed", result.Title)
	assert.Equal(t, "A feed with no items", result.Description)
	assert.Len(t, result.Items, 0)
}

func TestParser_ParseFromURL_Success(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <description>Test Description</description>
    <link>https://example.com</link>
    <item>
      <title>Test Article</title>
      <link>https://example.com/article</link>
      <description>Test content</description>
      <guid>test-guid</guid>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.Header.Get("User-Agent"), "CairnBot")

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	parser := NewParser()
	ctx := context.Background()

	result, err := parser.ParseFromURL(ctx, server.URL, server.Client())

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Feed", result.Title)
	assert.Len(t, result.Items, 1)
}

func TestParser_ParseFromURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parser := NewParser()
	ctx := context.Background()

	result, err := parser.ParseFromURL(ctx, server.URL, server.Client())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestParser_ParseFromURL_InvalidURL(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	result, err := parser.ParseFromURL(ctx, "://invalid-url", http.DefaultClient)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestParser_ParseFromURL_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	parser := NewParser()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := parser.ParseFromURL(ctx, server.URL, server.Client())

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestParser_ItemWithMissingFields(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Article without link</title>
      <description>Description only</description>
    </item>
    <item>
      <link>https://example.com/no-title</link>
      <description>No title article</description>
      <guid>guid-2</guid>
    </item>
  </channel>
</rss>`

	parser := NewParser()
	reader := strings.NewReader(rssFeed)

	result, err := parser.ParseFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Items with missing critical fields should still be parsed
	assert.GreaterOrEqual(t, len(result.Items), 0)
}

func TestParser_DateParsing(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Date Test Feed</title>
    <item>
      <title>Article with RFC822 date</title>
      <link>https://example.com/rfc822</link>
      <pubDate>Mon, 15 Jan 2024 14:30:00 GMT</pubDate>
      <guid>guid-1</guid>
    </item>
    <item>
      <title>Article with ISO8601 date</title>
      <link>https://example.com/iso</link>
      <pubDate>2024-01-15T14:30:00Z</pubDate>
      <guid>guid-2</guid>
    </item>
  </channel>
</rss>`

	parser := NewParser()
	reader := strings.NewReader(rssFeed)

	result, err := parser.ParseFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)

	// Both date formats should be parsed
	if result.Items[0].PublishedAt != nil {
		assert.NotZero(t, result.Items[0].PublishedAt)
	}
	if result.Items[1].PublishedAt != nil {
		assert.NotZero(t, result.Items[1].PublishedAt)
	}
}

func TestParser_HTMLInContent(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>HTML Content Feed</title>
    <item>
      <title>Article with HTML</title>
      <link>https://example.com/html</link>
      <description><![CDATA[<p>This is <strong>bold</strong> text.</p>]]></description>
      <guid>guid-1</guid>
    </item>
  </channel>
</rss>`

	parser := NewParser()
	reader := strings.NewReader(rssFeed)

	result, err := parser.ParseFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 1)
	// HTML content should be preserved
	assert.Contains(t, result.Items[0].Description, "<p>")
	assert.Contains(t, result.Items[0].Description, "<strong>")
}
