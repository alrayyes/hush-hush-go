package hushhush_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func TestClient_Retries503(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithMaxRetries(3))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := client.GetObject(context.Background(), "obj", "")
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if string(got) != "ok" {
		t.Errorf("body = %q, want %q", got, "ok")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_RetryAfterTakesPriority(t *testing.T) {
	var attempts int32
	var firstAttemptAt, secondAttemptAt time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			firstAttemptAt = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		secondAttemptAt = time.Now()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithMaxRetries(1))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.GetObject(context.Background(), "obj", ""); err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if elapsed := secondAttemptAt.Sub(firstAttemptAt); elapsed < time.Second {
		t.Errorf("retried after %v, want at least the 1s Retry-After", elapsed)
	}
}

func TestClient_DoesNotRetry400(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithMaxRetries(3))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetObject(context.Background(), "obj", "")
	if err == nil {
		t.Fatal("GetObject: want error, got nil")
	}
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *hushhush.APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (non-retryable status must not be retried)", attempts)
	}
}
