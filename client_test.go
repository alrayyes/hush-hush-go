package hushhush_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func TestNewClient_CredentialFromEnvironment(t *testing.T) {
	t.Setenv("HUSH_HUSH_API_KEY", "env-token")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "x"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.CreateObject(context.Background(), hushhush.CreateObjectRequest{Id: "x", Value: []byte("v")}, ""); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if want := "Bearer env-token"; gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
}

func TestNewClient_ExplicitCredentialOverridesEnvironment(t *testing.T) {
	t.Setenv("HUSH_HUSH_API_KEY", "env-token")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "x"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("explicit-token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.CreateObject(context.Background(), hushhush.CreateObjectRequest{Id: "x", Value: []byte("v")}, ""); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if want := "Bearer explicit-token"; gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
}

func TestClient_GetObject_UnauthenticatedReadSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("unexpected Authorization header on read: %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("sealed-bytes"))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL) // no credential set at all
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := client.GetObject(context.Background(), "my-object", "")
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if string(got) != "sealed-bytes" {
		t.Errorf("GetObject body = %q, want %q", got, "sealed-bytes")
	}
}

func TestClient_XCaller_IsPerRequest(t *testing.T) {
	var gotCaller string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCaller = r.Header.Get("X-Caller")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bytes"))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.GetObject(context.Background(), "obj", "caller-a"); err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if gotCaller != "caller-a" {
		t.Errorf("X-Caller = %q, want %q", gotCaller, "caller-a")
	}

	if _, err := client.GetObject(context.Background(), "obj", ""); err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if gotCaller != "" {
		t.Errorf("X-Caller = %q, want empty on a call with no caller set", gotCaller)
	}
}
