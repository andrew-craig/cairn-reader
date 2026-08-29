package sanitize_test

import (
	"os"
	"strings"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/rss/sanitize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- per-tag / per-attribute coverage ---

func TestSanitize_TextFormattingKept(t *testing.T) {
	in := `<p class="lead"><strong>Bold</strong> and <em>italic</em> and <code>code</code></p>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<p")
	assert.Contains(t, out, "<strong>")
	assert.Contains(t, out, "<em>")
	assert.Contains(t, out, "<code>")
}

func TestSanitize_ScriptStripped(t *testing.T) {
	in := `<p>Hello</p><script>alert('xss')</script>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "<script>")
	assert.NotContains(t, out, "alert")
	assert.Contains(t, out, "<p>Hello</p>")
}

func TestSanitize_StyleStripped(t *testing.T) {
	in := `<p>Text</p><style>body{color:red}</style>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "<style>")
	assert.NotContains(t, out, "color:red")
}

func TestSanitize_IframeStripped(t *testing.T) {
	in := `<p>Before</p><iframe src="https://evil.com"></iframe><p>After</p>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "<iframe")
	assert.NotContains(t, out, "evil.com")
}

func TestSanitize_FormStripped(t *testing.T) {
	in := `<form action="https://evil.com/steal"><input type="text"></form><p>ok</p>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "<form")
	assert.NotContains(t, out, "<input")
}

func TestSanitize_ObjectStripped(t *testing.T) {
	in := `<object data="bad.swf"></object><p>text</p>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "<object")
}

func TestSanitize_LinkKept(t *testing.T) {
	in := `<a href="https://example.com" title="Example">link text</a>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, `href="https://example.com"`)
	assert.Contains(t, out, `title="Example"`)
	assert.Contains(t, out, "link text")
}

func TestSanitize_JavascriptHrefStripped(t *testing.T) {
	in := `<a href="javascript:alert(1)">bad link</a>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "javascript:")
}

func TestSanitize_MailtoLinkKept(t *testing.T) {
	in := `<a href="mailto:help@example.com" title="Email">contact</a>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "mailto:help@example.com")
}

func TestSanitize_ImageKept(t *testing.T) {
	in := `<img src="https://example.com/img.png" alt="Alt text" title="Title" width="800" height="600">`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, `src="https://example.com/img.png"`)
	assert.Contains(t, out, `alt="Alt text"`)
	assert.Contains(t, out, `width="800"`)
}

func TestSanitize_ImageJavascriptSrcStripped(t *testing.T) {
	in := `<img src="javascript:alert(1)" alt="xss">`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, "javascript:")
}

func TestSanitize_FigureKept(t *testing.T) {
	in := `<figure><img src="https://example.com/p.jpg" alt="photo"><figcaption>Caption</figcaption></figure>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<figure>")
	assert.Contains(t, out, "<figcaption>")
}

func TestSanitize_AudioVideoSourceKept(t *testing.T) {
	in := `<audio controls><source src="https://cdn.example.com/ep1.mp3" type="audio/mpeg"></audio>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<audio")
	assert.Contains(t, out, "<source")
	assert.Contains(t, out, `src="https://cdn.example.com/ep1.mp3"`)
}

func TestSanitize_VideoKept(t *testing.T) {
	in := `<video src="https://cdn.example.com/clip.mp4" controls></video>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<video")
	assert.Contains(t, out, `src="https://cdn.example.com/clip.mp4"`)
}

func TestSanitize_PictureSourceSrcsetKept(t *testing.T) {
	in := `<picture><source srcset="hero.webp" media="(min-width:800px)" type="image/webp"><img src="hero.jpg" alt="hero"></picture>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<picture>")
	assert.Contains(t, out, "srcset=")
	assert.Contains(t, out, "media=")
}

func TestSanitize_TableKept(t *testing.T) {
	in := `<table><thead><tr><th scope="col">Name</th></tr></thead><tbody><tr><td colspan="2">Value</td></tr></tbody></table>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<table>")
	assert.Contains(t, out, `scope="col"`)
	assert.Contains(t, out, `colspan="2"`)
}

func TestSanitize_ClassOnParagraphKept(t *testing.T) {
	in := `<p class="lead intro">Hello</p>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, `class="lead intro"`)
}

func TestSanitize_ClassOnDisallowedElementStripped(t *testing.T) {
	// class is only allowed on p/div/span/blockquote/code/pre — not on <a>
	in := `<a href="https://example.com" class="btn">link</a>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, `class=`)
	assert.Contains(t, out, "link") // text preserved
}

func TestSanitize_IdOnHeadingKept(t *testing.T) {
	in := `<h2 id="section-two">Section Two</h2>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, `id="section-two"`)
}

func TestSanitize_IdWithWhitespaceStripped(t *testing.T) {
	// id containing whitespace violates HTML5 §2.7.1 — must be stripped
	in := `<h2 id="two words">Heading</h2>`
	out := sanitize.Sanitize(in)
	assert.NotContains(t, out, `id=`)
	assert.Contains(t, out, "Heading")
}

func TestSanitize_ListsKept(t *testing.T) {
	in := `<ul><li>One</li><li>Two</li></ul><ol><li>A</li></ol>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<ul>")
	assert.Contains(t, out, "<ol>")
	assert.Contains(t, out, "<li>")
}

func TestSanitize_CodePreKept(t *testing.T) {
	in := `<pre class="go"><code>package main</code></pre>`
	out := sanitize.Sanitize(in)
	assert.Contains(t, out, "<pre")
	assert.Contains(t, out, "<code>")
	assert.Contains(t, out, "package main")
}

func TestSanitize_EmptyString(t *testing.T) {
	assert.Equal(t, "", sanitize.Sanitize(""))
}

func TestSanitize_PlainText(t *testing.T) {
	out := sanitize.Sanitize("Hello, world!")
	assert.Equal(t, "Hello, world!", out)
}

// --- snapshot tests against real-world fixture files ---

type snapshotCase struct {
	fixture        string
	mustContain    []string
	mustNotContain []string
}

var snapshotCases = []snapshotCase{
	{
		fixture:        "testdata/blog_post.html",
		mustContain:    []string{"<h1", "Understanding Go Modules", "<p", "<code>", "<figure>", "<img", "<a href="},
		mustNotContain: []string{"<script>", "alert("},
	},
	{
		fixture:        "testdata/podcast_feed_item.html",
		mustContain:    []string{"<h2", "Episode 42", "<audio", "<source", `src="https://podcast.example.com/ep42.mp3"`, "<ol>", "<li>"},
		mustNotContain: []string{"<iframe", "evil.com"},
	},
	{
		fixture:        "testdata/responsive_images.html",
		mustContain:    []string{"<picture>", "<source", "srcset=", "media=", "<img", "<figure>"},
		mustNotContain: []string{"<style>", "color: red", "javascript:"},
	},
	{
		fixture:        "testdata/table_article.html",
		mustContain:    []string{"<table>", "<caption>", "<thead>", "<tbody>", `scope="col"`, `colspan="2"`},
		mustNotContain: []string{"<form", "<input"},
	},
	{
		fixture:        "testdata/code_article.html",
		mustContain:    []string{"<h1", "<pre", "<code>", "<kbd>", "<var>", "<del>", "<ins>", "mailto:"},
		mustNotContain: []string{"<object", "malware"},
	},
}

func TestSanitize_Snapshots(t *testing.T) {
	for _, tc := range snapshotCases {
		t.Run(tc.fixture, func(t *testing.T) {
			data, err := os.ReadFile(tc.fixture)
			require.NoError(t, err, "fixture file must exist")

			out := sanitize.Sanitize(string(data))
			require.NotEmpty(t, out)

			for _, want := range tc.mustContain {
				assert.True(t, strings.Contains(out, want),
					"expected %q to contain %q\n\noutput:\n%s", tc.fixture, want, out)
			}
			for _, banned := range tc.mustNotContain {
				assert.False(t, strings.Contains(out, banned),
					"expected %q NOT to contain %q\n\noutput:\n%s", tc.fixture, banned, out)
			}
		})
	}
}
