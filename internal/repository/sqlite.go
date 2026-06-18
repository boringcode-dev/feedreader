package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"feedreader/internal/domain"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) SaveSnapshot(source string, fetchedAt time.Time, items []domain.FeedItem) error {
	fetchedAt = fetchedAt.UTC()
	fetchedAtISO := toISO(&fetchedAt)
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range items {
		metadataJSON, err := json.Marshal(item.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		if _, err := tx.Exec(`
			INSERT INTO items (
				source, external_id, title, url, summary, author, score, comments_url,
				published_at, source_rank, metadata_json, first_seen_at, last_seen_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(source, external_id) DO UPDATE SET
				title = excluded.title,
				url = excluded.url,
				summary = excluded.summary,
				author = excluded.author,
				score = excluded.score,
				comments_url = excluded.comments_url,
				published_at = excluded.published_at,
				source_rank = excluded.source_rank,
				metadata_json = excluded.metadata_json,
				last_seen_at = excluded.last_seen_at,
				updated_at = excluded.updated_at
		`, item.Source, item.ExternalID, item.Title, item.URL, deref(item.Summary), deref(item.Author), intOrNil(item.Score), deref(item.CommentsURL), toISO(item.PublishedAt), item.SourceRank, string(metadataJSON), fetchedAtISO, fetchedAtISO, fetchedAtISO); err != nil {
			return fmt.Errorf("upsert item %s/%s: %w", item.Source, item.ExternalID, err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO sync_state (source, last_attempt_at, last_success_at, last_error, item_count)
		VALUES (?, ?, ?, NULL, ?)
		ON CONFLICT(source) DO UPDATE SET
			last_attempt_at = excluded.last_attempt_at,
			last_success_at = excluded.last_success_at,
			last_error = NULL,
			item_count = excluded.item_count
	`, source, fetchedAtISO, fetchedAtISO, len(items)); err != nil {
		return fmt.Errorf("upsert sync_state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) RecordFailure(source string, attemptedAt time.Time, message string) error {
	attemptedAt = attemptedAt.UTC()
	_, err := r.db.Exec(`
		INSERT INTO sync_state (source, last_attempt_at, last_success_at, last_error, item_count)
		VALUES (?, ?, NULL, ?, 0)
		ON CONFLICT(source) DO UPDATE SET
			last_attempt_at = excluded.last_attempt_at,
			last_error = excluded.last_error
	`, source, toISO(&attemptedAt), truncate(message, 500))
	if err != nil {
		return fmt.Errorf("record failure: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListSourceStates() (map[string]domain.SyncState, error) {
	rows, err := r.db.Query(`SELECT source, last_attempt_at, last_success_at, last_error, item_count FROM sync_state`)
	if err != nil {
		return nil, fmt.Errorf("query sync_state: %w", err)
	}
	defer rows.Close()

	result := map[string]domain.SyncState{}
	for rows.Next() {
		var source string
		var lastAttempt, lastSuccess, lastError sql.NullString
		var itemCount int
		if err := rows.Scan(&source, &lastAttempt, &lastSuccess, &lastError, &itemCount); err != nil {
			return nil, fmt.Errorf("scan sync_state: %w", err)
		}
		result[source] = domain.SyncState{
			Source:        source,
			LastAttemptAt: fromNullString(lastAttempt),
			LastSuccessAt: fromNullString(lastSuccess),
			LastError:     stringPtr(lastError),
			ItemCount:     itemCount,
		}
	}
	return result, rows.Err()
}

func (r *SQLiteRepository) GetCurrentItems(source string, limit int) ([]domain.FeedItem, error) {
	rows, err := r.db.Query(`
		SELECT source, external_id, title, url, summary, author, score, comments_url, published_at, source_rank, metadata_json
		FROM items
		WHERE source = ?
		  AND last_seen_at = (
		      SELECT last_success_at FROM sync_state WHERE source = ?
		  )
		ORDER BY source_rank ASC
		LIMIT ?
	`, source, source, limit)
	if err != nil {
		return nil, fmt.Errorf("query current items: %w", err)
	}
	defer rows.Close()

	var items []domain.FeedItem
	for rows.Next() {
		item, err := scanFeedItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) ListFeedItems(limit int, offset int, source string, searchQuery string) ([]domain.FeedItem, error) {
	query := `
		SELECT source, external_id, title, url, summary, author, score, comments_url, published_at, source_rank, metadata_json
		FROM items
	`
	args := []any{}
	conditions := make([]string, 0, 1+len(searchTerms(searchQuery)))
	if strings.TrimSpace(source) != "" {
		conditions = append(conditions, `source = ?`)
		args = append(args, source)
	}
	for _, term := range searchTerms(searchQuery) {
		conditions = append(conditions, `(
			lower(title) LIKE ? ESCAPE '\'
			OR lower(coalesce(summary, '')) LIKE ? ESCAPE '\'
			OR lower(coalesce(author, '')) LIKE ? ESCAPE '\'
			OR lower(url) LIKE ? ESCAPE '\'
			OR lower(coalesce(metadata_json, '')) LIKE ? ESCAPE '\'
		)`)
		pattern := likePattern(term)
		for range 5 {
			args = append(args, pattern)
		}
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	query += ` ORDER BY (published_at IS NULL) ASC, published_at DESC, updated_at DESC, first_seen_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	if offset > 0 {
		query += ` OFFSET ?`
		args = append(args, offset)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query feed items: %w", err)
	}
	defer rows.Close()

	items := []domain.FeedItem{}
	for rows.Next() {
		item, err := scanFeedItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) CountTotalItems() (int, error) {
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count items: %w", err)
	}
	return total, nil
}

func scanFeedItem(scanner interface{ Scan(dest ...any) error }) (domain.FeedItem, error) {
	var source, externalID, title, url string
	var summary, author, commentsURL, publishedAt, metadataJSON sql.NullString
	var score sql.NullInt64
	var sourceRank int
	if err := scanner.Scan(&source, &externalID, &title, &url, &summary, &author, &score, &commentsURL, &publishedAt, &sourceRank, &metadataJSON); err != nil {
		return domain.FeedItem{}, fmt.Errorf("scan item: %w", err)
	}
	metadata := map[string]any{}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return domain.FeedItem{}, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return domain.FeedItem{
		Source:      source,
		ExternalID:  externalID,
		Title:       title,
		URL:         url,
		Summary:     stringPtr(summary),
		Author:      stringPtr(author),
		Score:       intPtr(score),
		CommentsURL: stringPtr(commentsURL),
		PublishedAt: fromNullString(publishedAt),
		SourceRank:  sourceRank,
		Metadata:    metadata,
	}, nil
}

func toISO(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func fromNullString(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func stringPtr(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	copy := value.String
	return &copy
}

func intPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	copy := int(value.Int64)
	return &copy
}

func intOrNil(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func deref(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func searchTerms(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	terms := strings.Fields(strings.ToLower(trimmed))
	filtered := make([]string, 0, len(terms))
	for _, term := range terms {
		if term != "" {
			filtered = append(filtered, term)
		}
	}
	return filtered
}

func likePattern(term string) string {
	replacer := strings.NewReplacer(`\\`, `\\\\`, `%`, `\\%`, `_`, `\\_`)
	return "%" + replacer.Replace(term) + "%"
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
