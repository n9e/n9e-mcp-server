package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// APIError represents Nightingale API error with complete request context
type APIError struct {
	Method     string     // HTTP method: GET/POST/PUT/DELETE
	Path       string     // Request path: /api/n9e/alert-cur-events/list
	Params     url.Values // GET query parameters
	Body       any        // POST/PUT request body (only key fields are recorded to avoid sensitive info)
	StatusCode int        // HTTP status code
	ErrMsg     string     // Error message from Nightingale err field
	RequestID  string     // Request ID (if present in response header)
}

func (e *APIError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("n9e api error: %s %s", e.Method, e.Path))
	sb.WriteString(fmt.Sprintf(" status=%d", e.StatusCode))
	sb.WriteString(fmt.Sprintf(" err=%q", e.ErrMsg))

	if len(e.Params) > 0 {
		sb.WriteString(fmt.Sprintf(" params=%v", e.Params))
	}
	if e.Body != nil {
		// Only output summary to avoid large logs
		bodyJSON, _ := json.Marshal(e.Body)
		if len(bodyJSON) > 200 {
			sb.WriteString(fmt.Sprintf(" body=%s...(truncated)", bodyJSON[:200]))
		} else {
			sb.WriteString(fmt.Sprintf(" body=%s", bodyJSON))
		}
	}
	if e.RequestID != "" {
		sb.WriteString(fmt.Sprintf(" request_id=%s", e.RequestID))
	}

	return sb.String()
}

// IsAPIError checks if the error is an API business error
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

// GetAPIError extracts APIError (for getting detailed information)
func GetAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}
