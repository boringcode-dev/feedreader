package domain

import "time"

type FeedItem struct {
	Source      string
	ExternalID  string
	Title       string
	URL         string
	Summary     *string
	Author      *string
	Score       *int
	CommentsURL *string
	PublishedAt *time.Time
	FetchedAt   *time.Time
	SourceRank  int
	Metadata    map[string]any
}

type SyncState struct {
	Source        string
	LastAttemptAt *time.Time
	LastSuccessAt *time.Time
	LastError     *string
	ItemCount     int
}

type SourceSnapshot struct {
	Source        string
	Label         string
	HomepageURL   string
	LastAttemptAt *time.Time
	LastSuccessAt *time.Time
	LastError     *string
	ItemCount     int
	Items         []FeedItem
}

type RefreshOutcome struct {
	Source    string `json:"source"`
	OK        bool   `json:"ok"`
	ItemCount int    `json:"item_count"`
	Error     string `json:"error,omitempty"`
}

type CardView struct {
	Source        string
	Index         int
	Title         string
	URL           string
	Brief         *string
	BriefPrefix   *string
	BriefSuffix   *string
	BriefDateISO  *string
	BriefDateKind string
	Host          string
}

type ErrorView struct {
	Source string
	Label  string
	Error  string
}
