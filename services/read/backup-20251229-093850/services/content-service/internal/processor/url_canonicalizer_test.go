package processor

import (
	"testing"
)

func TestURLCanonicalizer_Canonicalize(t *testing.T) {
	canonicalizer := NewURLCanonicalizer()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "basic URL with tracking params",
			input:   "https://example.com/article?utm_source=twitter&utm_medium=social",
			want:    "https://example.com/article",
			wantErr: false,
		},
		{
			name:    "URL with fbclid",
			input:   "https://example.com/page?fbclid=12345",
			want:    "https://example.com/page",
			wantErr: false,
		},
		{
			name:    "URL with mixed params",
			input:   "https://example.com/page?id=123&utm_campaign=email&ref=link",
			want:    "https://example.com/page?id=123",
			wantErr: false,
		},
		{
			name:    "URL with trailing slash",
			input:   "https://example.com/article/",
			want:    "https://example.com/article",
			wantErr: false,
		},
		{
			name:    "root URL keeps trailing slash",
			input:   "https://example.com/",
			want:    "https://example.com/",
			wantErr: false,
		},
		{
			name:    "uppercase host normalized",
			input:   "https://EXAMPLE.COM/Article",
			want:    "https://example.com/Article",
			wantErr: false,
		},
		{
			name:    "URL with no path",
			input:   "https://example.com",
			want:    "https://example.com/",
			wantErr: false,
		},
		{
			name:    "URL with fragment",
			input:   "https://example.com/page#section",
			want:    "https://example.com/page#section",
			wantErr: false,
		},
		{
			name:    "multiple tracking params",
			input:   "https://example.com/article?utm_source=google&utm_medium=cpc&utm_campaign=spring&gclid=abc123",
			want:    "https://example.com/article",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalizer.Canonicalize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Canonicalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Canonicalize() = %v, want %v", got, tt.want)
			}
		})
	}
}
