// Package api calls the Mercy Pay REST API: the key, retries and the error shape.
package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is where the API lives.
const DefaultBaseURL = "https://mercy-pay.com"

// Client sends requests with one key. The zero value is not usable; use New.
type Client struct {
	BaseURL    string
	APIKey     string
	UserAgent  string
	HTTP       *http.Client
	MaxRetries int
	// Sleep waits between two attempts; replaced in tests.
	Sleep func(time.Duration)
}

// New returns a client with the defaults: 30 s timeout, 2 retries.
func New(baseURL, apiKey, userAgent string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		UserAgent:  userAgent,
		HTTP:       &http.Client{Timeout: 30 * time.Second},
		MaxRetries: 2,
		Sleep:      time.Sleep,
	}
}

// Response is what the API answered.
type Response struct {
	Status    int
	Body      []byte
	RequestID string
	Header    http.Header
}

// Error is the API's {"error": {...}} body.
type Error struct {
	Status    int    `json:"-"`
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Param     string `json:"param"`
	RequestID string `json:"request_id"`
}

func (e *Error) Error() string {
	text := fmt.Sprintf("%d %s : %s", e.Status, e.Code, e.Message)
	if e.Param != "" {
		text += fmt.Sprintf(" (%s)", e.Param)
	}
	return text
}

// Request describes one call. Path may start with /v1/ or /api/v1/.
type Request struct {
	Method         string
	Path           string
	Query          url.Values
	Body           any
	IdempotencyKey string
}

// Do sends the request. A POST always carries an Idempotency-Key (generated once when none is
// given), so retrying it after a network error or a 5xx can never create twice. An answer of
// 400 or more comes back as both the Response and an *Error.
func (c *Client) Do(ctx context.Context, request Request) (*Response, error) {
	method := strings.ToUpper(request.Method)
	if method == http.MethodPost && request.IdempotencyKey == "" {
		request.IdempotencyKey = newKey()
	}

	var payload []byte
	if request.Body != nil {
		encoded, err := json.Marshal(request.Body)
		if err != nil {
			return nil, err
		}
		payload = encoded
	}

	target := c.BaseURL + normalizePath(request.Path)
	if len(request.Query) > 0 {
		target += "?" + request.Query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			c.Sleep(backoff(attempt, lastErr))
		}

		response, err := c.send(ctx, method, target, payload, request.IdempotencyKey)
		if err != nil {
			if ctx.Err() != nil {
				return nil, err
			}
			lastErr = &retryable{err: err}
			continue
		}

		if response.Status == http.StatusTooManyRequests || response.Status >= 500 {
			lastErr = &retryable{err: asError(response), after: retryAfter(response.Header)}
			if attempt < c.MaxRetries {
				continue
			}
		}

		if response.Status >= 400 {
			return response, asError(response)
		}
		return response, nil
	}

	var wrapped *retryable
	if errors.As(lastErr, &wrapped) {
		return nil, wrapped.err
	}
	return nil, lastErr
}

func (c *Client) send(ctx context.Context, method, target string, payload []byte, idempotencyKey string) (*Response, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("User-Agent", c.UserAgent)
	if payload != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if idempotencyKey != "" {
		httpRequest.Header.Set("Idempotency-Key", idempotencyKey)
	}

	httpResponse, err := c.HTTP.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	content, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	return &Response{
		Status:    httpResponse.StatusCode,
		Body:      content,
		RequestID: httpResponse.Header.Get("Request-Id"),
		Header:    httpResponse.Header,
	}, nil
}

type retryable struct {
	err   error
	after time.Duration
}

func (r *retryable) Error() string { return r.err.Error() }
func (r *retryable) Unwrap() error { return r.err }

func backoff(attempt int, last error) time.Duration {
	var wrapped *retryable
	if errors.As(last, &wrapped) && wrapped.after > 0 {
		return wrapped.after
	}
	return time.Duration(500*(1<<(attempt-1))) * time.Millisecond
}

func retryAfter(header http.Header) time.Duration {
	seconds, err := strconv.Atoi(header.Get("Retry-After"))
	if err != nil || seconds < 0 {
		return 0
	}
	if seconds > 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func asError(response *Response) *Error {
	var envelope struct {
		Error Error `json:"error"`
	}
	_ = json.Unmarshal(response.Body, &envelope)
	apiError := envelope.Error
	apiError.Status = response.Status
	if apiError.Code == "" {
		apiError.Code = "http_" + strconv.Itoa(response.Status)
	}
	if apiError.Message == "" {
		apiError.Message = http.StatusText(response.Status)
	}
	if apiError.RequestID == "" {
		apiError.RequestID = response.RequestID
	}
	return &apiError
}

func normalizePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.HasPrefix(path, "/v1/") || path == "/v1" {
		path = "/api" + path
	}
	return path
}

func newKey() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return "cli_" + hex.EncodeToString(raw)
}
