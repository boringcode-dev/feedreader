package sources

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGetWithRetryRetriesTransportError(t *testing.T) {
	original := upstreamRetryDelays
	upstreamRetryDelays = []time.Duration{0, 0}
	defer func() { upstreamRetryDelays = original }()

	attempts := 0
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("connection reset by peer")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})}

	resp, err := getWithRetry(context.Background(), client, "https://example.com")
	if err != nil {
		t.Fatalf("getWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestGetWithRetryRetriesRetryableStatus(t *testing.T) {
	original := upstreamRetryDelays
	upstreamRetryDelays = []time.Duration{0}
	defer func() { upstreamRetryDelays = original }()

	attempts := 0
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		status := http.StatusBadGateway
		if attempts == 2 {
			status = http.StatusOK
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(http.StatusText(status))),
			Header:     make(http.Header),
		}, nil
	})}

	resp, err := getWithRetry(context.Background(), client, "https://example.com")
	if err != nil {
		t.Fatalf("getWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestGetWithRetryDoesNotRetryNonRetryableStatus(t *testing.T) {
	original := upstreamRetryDelays
	upstreamRetryDelays = []time.Duration{time.Hour}
	defer func() { upstreamRetryDelays = original }()

	attempts := 0
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("missing")),
			Header:     make(http.Header),
		}, nil
	})}

	resp, err := getWithRetry(context.Background(), client, "https://example.com")
	if err != nil {
		t.Fatalf("getWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}
