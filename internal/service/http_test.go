package service

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestUserAgentTransportSetsConfiguredUserAgent(t *testing.T) {
	transport := &userAgentTransport{
		base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("User-Agent"); got != "feedreader/0.1" {
				t.Fatalf("unexpected user-agent: %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
		userAgent: "feedreader/0.1",
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	_ = resp.Body.Close()
}

func TestUserAgentTransportPreservesExplicitUserAgent(t *testing.T) {
	transport := &userAgentTransport{
		base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("User-Agent"); got != "custom-agent/1.0" {
				t.Fatalf("unexpected user-agent: %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
		userAgent: "feedreader/0.1",
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("User-Agent", "custom-agent/1.0")
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	_ = resp.Body.Close()
}
