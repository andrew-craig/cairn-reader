package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFeedList(t *testing.T) {
	t.Parallel()

	content := `# header comment
https://example.com/a.xml
  https://example.com/b.xml
# another comment

https://example.com/c.xml
ftp://example.com/d.xml
not a url
`
	got := parseFeedList(content)
	want := []string{
		"https://example.com/a.xml",
		"https://example.com/b.xml",
		"https://example.com/c.xml",
	}

	if len(got) != len(want) {
		t.Fatalf("parseFeedList returned %d urls, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseFeedList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadFeedList_PrefersLocalFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "feeds.txt")
	fileContents := "https://file.example.com/feed.xml\n"
	if err := os.WriteFile(path, []byte(fileContents), 0o600); err != nil {
		t.Fatalf("write temp feed list: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("loadFeedList should not fall back to URL when file exists")
		_, _ = w.Write([]byte("https://url.example.com/feed.xml"))
	}))
	defer server.Close()

	s := NewFeedSyncer(nil, path, server.URL)

	body, source, err := s.loadFeedList(context.Background())
	if err != nil {
		t.Fatalf("loadFeedList: %v", err)
	}
	if string(body) != fileContents {
		t.Errorf("loadFeedList body = %q, want %q", body, fileContents)
	}
	if source != "file:"+path {
		t.Errorf("loadFeedList source = %q, want file:%s", source, path)
	}
}

func TestLoadFeedList_FallsBackToURL(t *testing.T) {
	t.Parallel()

	urlBody := "https://url.example.com/feed.xml\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(urlBody))
	}))
	defer server.Close()

	// Non-existent path must trigger the URL fallback, not an error.
	missing := filepath.Join(t.TempDir(), "does-not-exist.txt")
	s := NewFeedSyncer(nil, missing, server.URL)

	body, source, err := s.loadFeedList(context.Background())
	if err != nil {
		t.Fatalf("loadFeedList: %v", err)
	}
	if string(body) != urlBody {
		t.Errorf("loadFeedList body = %q, want %q", body, urlBody)
	}
	if source != "url:"+server.URL {
		t.Errorf("loadFeedList source = %q, want url:%s", source, server.URL)
	}
}

func TestLoadFeedList_NoSourcesConfigured(t *testing.T) {
	t.Parallel()

	s := NewFeedSyncer(nil, "", "")
	if _, _, err := s.loadFeedList(context.Background()); err == nil {
		t.Fatal("expected error when neither path nor URL is configured")
	}
}

func TestLoadFeedList_URLErrorStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	s := NewFeedSyncer(nil, "", server.URL)
	if _, _, err := s.loadFeedList(context.Background()); err == nil {
		t.Fatal("expected error when URL returns 500")
	}
}
