package sources

import "fmt"

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
