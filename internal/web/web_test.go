package web

import "testing"

func TestBuildEmptyMessage(t *testing.T) {
	tests := []struct {
		name   string
		source string
		query  string
		want   string
	}{
		{
			name: "default empty message",
			want: "No items yet. The scheduler will populate the feed automatically.",
		},
		{
			name:  "search empty message",
			query: "agents",
			want:  "No matches found. Try a different query.",
		},
		{
			name:   "filtered empty message",
			source: "github",
			want:   "No items found in GitHub Trending right now.",
		},
		{
			name:   "filtered search empty message",
			source: "alphaxiv",
			query:  "vision",
			want:   "No matches found in alphaXiv. Try a different query.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildEmptyMessage(tt.source, tt.query); got != tt.want {
				t.Fatalf("buildEmptyMessage(%q, %q) = %q, want %q", tt.source, tt.query, got, tt.want)
			}
		})
	}
}
