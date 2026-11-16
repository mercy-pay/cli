package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(server *httptest.Server) *Client {
	client := New(server.URL, "mp_sandbox_key", "mercypay-cli/test")
	client.Sleep = func(time.Duration) {}
	return client
}

func TestSendsKeyAndNormalizesPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/payments" || r.URL.Query().Get("limit") != "2" {
			t.Errorf("unexpected %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer mp_sandbox_key" {
			t.Errorf("missing key")
		}
		w.Header().Set("Request-Id", "req_1")
		w.Write([]byte(`{"object":"list"}`))
	}))
	defer server.Close()

	response, err := testClient(server).Do(context.Background(), Request{Method: "GET", Path: "/v1/payments", Query: map[string][]string{"limit": {"2"}}})
	if err != nil || response.RequestID != "req_1" {
		t.Fatalf("%v %+v", err, response)
	}
}

func TestRetriesAPostWithTheSameIdempotencyKey(t *testing.T) {
	var calls atomic.Int32
	keys := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys[r.Header.Get("Idempotency-Key")] = true
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	response, err := testClient(server).Do(context.Background(), Request{Method: "POST", Path: "/v1/payouts", Body: map[string]any{"amount": 1}})
	if err != nil || response.Status != http.StatusCreated {
		t.Fatalf("%v %+v", err, response)
	}
	if calls.Load() != 3 || len(keys) != 1 {
		t.Fatalf("calls=%d keys=%v", calls.Load(), keys)
	}
}

func TestReturnsTheApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error":{"type":"invalid_request_error","code":"validation_failed","message":"Montant invalide.","param":"amount","request_id":"req_9"}}`))
	}))
	defer server.Close()

	_, err := testClient(server).Do(context.Background(), Request{Method: "POST", Path: "/v1/payments"})
	var apiError *Error
	if !errors.As(err, &apiError) || apiError.Code != "validation_failed" || apiError.Param != "amount" || apiError.Status != 422 {
		t.Fatalf("got %v", err)
	}
}

func TestGivesUpAfterTheRetries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	response, err := testClient(server).Do(context.Background(), Request{Method: "GET", Path: "/v1/account"})
	var apiError *Error
	if !errors.As(err, &apiError) || apiError.Status != 429 || response == nil || calls.Load() != 3 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}
