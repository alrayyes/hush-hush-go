package hushhush

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError represents a non-2xx response from hush-hush. It's returned
// instead of a bare string or the raw *http.Response so callers can branch
// on the status code and inspect the request ID without parsing anything
// themselves.
type APIError struct {
	// StatusCode is the HTTP status hush-hush responded with.
	StatusCode int
	// RequestID is populated when the response carries a documented
	// request-ID header; hush-hush's spec doesn't currently document one,
	// so this is usually empty. Left as a field rather than omitted so a
	// future spec addition doesn't change the error type's shape.
	RequestID string
	// Message is the parsed `error` field from hush-hush's error body, or
	// empty if the body wasn't the expected shape.
	Message string
	// Body is the raw, unparsed response body, for a caller that needs
	// more than Message.
	Body []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("hushhush: %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("hushhush: unexpected status %d", e.StatusCode)
}

// newAPIError builds an APIError from a response's status, headers and body.
// Never fails: a body that isn't the documented {"error": "..."} shape just
// leaves Message empty rather than returning a second error to handle.
func newAPIError(statusCode int, header http.Header, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		RequestID:  header.Get("X-Request-Id"),
		Body:       body,
	}
	var parsed Error
	if json.Unmarshal(body, &parsed) == nil {
		apiErr.Message = parsed.Error
	}
	return apiErr
}
