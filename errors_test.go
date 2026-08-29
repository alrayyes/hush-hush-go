package hushhush_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func TestClient_TypedErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"unknown object"}`))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetObject(context.Background(), "missing", "")
	if err == nil {
		t.Fatal("GetObject: want error, got nil")
	}
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *hushhush.APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if apiErr.Message != "unknown object" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "unknown object")
	}
	if apiErr.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", apiErr.RequestID, "req-123")
	}
}

func TestClient_TypedErrorMapping_NoRequestIDHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"unknown object"}`))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetObject(context.Background(), "missing", "")
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *hushhush.APIError", err)
	}
	if apiErr.RequestID != "" {
		t.Errorf("RequestID = %q, want empty when the server sends no request-ID header", apiErr.RequestID)
	}
}
