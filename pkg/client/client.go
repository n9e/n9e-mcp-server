package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/n9e/n9e-mcp-server/pkg/types"
)

const (
	DefaultTimeout    = 30 * time.Second
	DefaultMaxRetries = 3
	DefaultRetryDelay = 1 * time.Second
	maxResponseSize   = 10 * 1024 * 1024 // 10MB
)

// Client is the Nightingale API client
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string
	userAgent  string
}

// NewClient creates a Nightingale API client
func NewClient(token, baseURL, userAgent string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	if baseURL == "" {
		baseURL = "http://localhost:17000"
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:   parsedURL,
		token:     token,
		userAgent: userAgent,
	}, nil
}

// SetUserAgent sets the User-Agent
func (c *Client) SetUserAgent(userAgent string) {
	c.userAgent = userAgent
}

// doRequest executes HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	// Build complete URL
	fullURL := c.baseURL.ResolveReference(&url.URL{Path: path})
	if params != nil {
		fullURL.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-User-Token", c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	return c.httpClient.Do(req)
}

// makeRequest is the request method with timeout and retry
func (c *Client) makeRequest(ctx context.Context, method, path string, params url.Values, body any) ([]byte, int, string, error) {
	var lastErr error

	for attempt := 0; attempt <= DefaultMaxRetries; attempt++ {
		// Check if context is cancelled/timed out
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, 0, "", fmt.Errorf("request canceled: %w", err)
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, 0, "", fmt.Errorf("request timeout: %w", err)
			}
			return nil, 0, "", err
		}

		resp, err := c.doRequest(ctx, method, path, params, body)
		if err != nil {
			lastErr = err
			if isRetryableError(err) && attempt < DefaultMaxRetries {
				time.Sleep(retryDelay(attempt))
				continue
			}
			return nil, 0, "", fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to read response: %w", err)
		}

		requestID := resp.Header.Get("X-Request-Id")

		// Decide whether to retry based on status code
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return bodyBytes, resp.StatusCode, requestID, nil

		case resp.StatusCode == 429: // Too Many Requests
			if attempt < DefaultMaxRetries {
				delay := parseRetryAfter(resp.Header)
				if delay == 0 {
					delay = retryDelay(attempt)
				}
				time.Sleep(delay)
				continue
			}
			return nil, resp.StatusCode, requestID, fmt.Errorf("rate limited (429), retries exhausted")

		case resp.StatusCode >= 500: // 5xx server errors are retryable
			lastErr = fmt.Errorf("server error: %d %s", resp.StatusCode, string(bodyBytes))
			if attempt < DefaultMaxRetries {
				time.Sleep(retryDelay(attempt))
				continue
			}
			return nil, resp.StatusCode, requestID, lastErr

		case resp.StatusCode >= 400: // 4xx client errors are not retryable
			return nil, resp.StatusCode, requestID, fmt.Errorf("client error: %d %s", resp.StatusCode, string(bodyBytes))

		default:
			return nil, resp.StatusCode, requestID, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}
	}

	return nil, 0, "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// retryDelay calculates exponential backoff delay
func retryDelay(attempt int) time.Duration {
	delay := DefaultRetryDelay * time.Duration(1<<attempt)
	// Add jitter to avoid thundering herd
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	return delay + jitter
}

// isRetryableError checks if the error is retryable
func isRetryableError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.Temporary() {
		return true
	}
	return false
}

// parseRetryAfter parses Retry-After header
func parseRetryAfter(header http.Header) time.Duration {
	if v := header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

// DoGet executes GET request
func DoGet[T any](c *Client, ctx context.Context, path string, params url.Values) (T, error) {
	var zero T

	bodyBytes, httpStatus, requestID, err := c.makeRequest(ctx, "GET", path, params, nil)
	if err != nil {
		return zero, err
	}

	var resp types.N9eResponse[T]
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		// Provide detailed error info for diagnosis
		preview := string(bodyBytes)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return zero, fmt.Errorf("failed to unmarshal response (check N9E_BASE_URL and N9E_TOKEN): %w, response preview: %s", err, preview)
	}

	// Check business error
	if resp.Err != "" {
		return zero, &APIError{
			Method:     "GET",
			Path:       path,
			Params:     params,
			StatusCode: httpStatus,
			ErrMsg:     resp.Err,
			RequestID:  requestID,
		}
	}

	return resp.Dat, nil
}

// DoPost executes POST request
func DoPost[T any](c *Client, ctx context.Context, path string, body any) (T, error) {
	var zero T

	bodyBytes, httpStatus, requestID, err := c.makeRequest(ctx, "POST", path, nil, body)
	if err != nil {
		return zero, err
	}

	var resp types.N9eResponse[T]
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		preview := string(bodyBytes)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return zero, fmt.Errorf("failed to unmarshal response (check N9E_BASE_URL and N9E_TOKEN): %w, response preview: %s", err, preview)
	}

	if resp.Err != "" {
		return zero, &APIError{
			Method:     "POST",
			Path:       path,
			Body:       body,
			StatusCode: httpStatus,
			ErrMsg:     resp.Err,
			RequestID:  requestID,
		}
	}

	return resp.Dat, nil
}

// DoPut executes PUT request
func DoPut[T any](c *Client, ctx context.Context, path string, body any) (T, error) {
	var zero T

	bodyBytes, httpStatus, requestID, err := c.makeRequest(ctx, "PUT", path, nil, body)
	if err != nil {
		return zero, err
	}

	var resp types.N9eResponse[T]
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		preview := string(bodyBytes)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return zero, fmt.Errorf("failed to unmarshal response (check N9E_BASE_URL and N9E_TOKEN): %w, response preview: %s", err, preview)
	}

	if resp.Err != "" {
		return zero, &APIError{
			Method:     "PUT",
			Path:       path,
			Body:       body,
			StatusCode: httpStatus,
			ErrMsg:     resp.Err,
			RequestID:  requestID,
		}
	}

	return resp.Dat, nil
}

// DoDelete executes DELETE request
func DoDelete[T any](c *Client, ctx context.Context, path string, body any) (T, error) {
	var zero T

	bodyBytes, httpStatus, requestID, err := c.makeRequest(ctx, "DELETE", path, nil, body)
	if err != nil {
		return zero, err
	}

	var resp types.N9eResponse[T]
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		preview := string(bodyBytes)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return zero, fmt.Errorf("failed to unmarshal response (check N9E_BASE_URL and N9E_TOKEN): %w, response preview: %s", err, preview)
	}

	if resp.Err != "" {
		return zero, &APIError{
			Method:     "DELETE",
			Path:       path,
			Body:       body,
			StatusCode: httpStatus,
			ErrMsg:     resp.Err,
			RequestID:  requestID,
		}
	}

	return resp.Dat, nil
}
