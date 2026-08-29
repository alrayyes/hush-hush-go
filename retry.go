package hushhush

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// retryTransport retries a request on network failure or an HTTP 5xx/429
// response, using exponential backoff with jitter, and honors a
// Retry-After response header ahead of its own backoff schedule when
// present. It never retries any other 4xx — those won't succeed on a
// second attempt, and retrying only delays the real error reaching the
// caller.
type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(attempt, lastResp)
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(delay):
			}
		}

		attemptReq := req
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptReq = req.Clone(req.Context())
			attemptReq.Body = body
		}

		resp, err := t.base.RoundTrip(attemptReq)
		if err != nil {
			lastErr, lastResp = err, nil
			continue
		}

		if !isRetryableStatus(resp.StatusCode) || attempt == t.maxRetries {
			return resp, nil
		}

		// Retrying: drain and close this attempt's body so its connection
		// can be reused, then fall through to the backoff/retry above.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		lastResp, lastErr = resp, nil
	}

	return lastResp, lastErr
}

func isRetryableStatus(status int) bool {
	return status >= 500 || status == http.StatusTooManyRequests
}

// backoffDelay honors resp's Retry-After header when present, ahead of the
// computed exponential-with-jitter delay.
func backoffDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if d, ok := retryAfterDelay(resp.Header.Get("Retry-After")); ok {
			return d
		}
	}
	base := 100 * time.Millisecond * time.Duration(1<<uint(attempt-1))
	jitter := time.Duration(rand.Int63n(int64(base) + 1)) //nolint:gosec // timing jitter, not security-sensitive
	return base + jitter
}

func retryAfterDelay(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := time.Until(t); d > 0 {
			return d, true
		}
		return 0, true
	}
	return 0, false
}
