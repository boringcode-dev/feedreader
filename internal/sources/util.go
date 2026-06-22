package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var upstreamRetryDelays = []time.Duration{250 * time.Millisecond, 750 * time.Millisecond}

type httpError struct {
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("unexpected status %d", e.StatusCode)
	}
	return fmt.Sprintf("unexpected status %d: %s", e.StatusCode, e.Body)
}

func getWithRetry(ctx context.Context, client *http.Client, rawURL string) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}
	attempts := len(upstreamRetryDelays) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil {
			if !shouldRetryStatus(resp.StatusCode) || attempt == attempts {
				return resp, nil
			}
			drainAndClose(resp.Body)
		} else if ctx.Err() != nil || attempt == attempts {
			return nil, err
		}
		if err := sleepWithContext(ctx, upstreamRetryDelay(attempt)); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("upstream retry loop exhausted for %s", rawURL)
}

func upstreamRetryDelay(attempt int) time.Duration {
	if attempt <= 0 || attempt > len(upstreamRetryDelays) {
		return 0
	}
	return upstreamRetryDelays[attempt-1]
}

func shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4096))
	_ = body.Close()
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func extractInt(value, pattern string) *int {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(value)
	if len(match) != 2 {
		return nil
	}
	parsed := strings.ReplaceAll(match[1], ",", "")
	if parsed == "" {
		return nil
	}
	out, err := strconv.Atoi(parsed)
	if err != nil {
		return nil
	}
	return &out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cleanString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
