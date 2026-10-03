package hushhush_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

func TestNewClient_CredentialFromEnvironment(t *testing.T) {
	t.Setenv("HUSH_HUSH_API_KEY", "env-token")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"slug": "x"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.CreateObject(context.Background(), hushhush.CreateObjectRequest{Slug: "x", Value: []byte("v")}, ""); err != nil {
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
		_ = json.NewEncoder(w).Encode(map[string]any{"slug": "x"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("explicit-token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.CreateObject(context.Background(), hushhush.CreateObjectRequest{Slug: "x", Value: []byte("v")}, ""); err != nil {
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

func TestClient_AuthStatus_UnauthenticatedSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("unexpected Authorization header on AuthStatus: %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.AuthStatus{Bootstrapped: true})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL) // no credential set at all
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	status, err := client.AuthStatus(context.Background())
	if err != nil {
		t.Fatalf("AuthStatus: %v", err)
	}
	if !status.Bootstrapped {
		t.Errorf("AuthStatus.Bootstrapped = false, want true")
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

func TestClient_ListObjects_RequiresCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "missing or invalid bearer token"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]hushhush.ObjectMetadata{{Slug: "obj-1"}})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.ListObjects(context.Background(), ""); err == nil {
		t.Error("ListObjects with no credential: want error, got nil")
	}

	client, err = hushhush.NewClient(srv.URL, hushhush.WithAPIKey("secret-token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	objects, err := client.ListObjects(context.Background(), "")
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(objects) != 1 || objects[0].Slug != "obj-1" {
		t.Errorf("objects = %+v, want one entry for obj-1", objects)
	}
}

func TestClient_ListObjects_UsedByFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]hushhush.ObjectMetadata{})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.ListObjects(context.Background(), "homelab/vps-docker"); err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if gotQuery != "used_by=homelab%2Fvps-docker" {
		t.Errorf("query = %q, want a used_by filter, got %q", gotQuery, gotQuery)
	}
}

func TestClient_GetOwnerIdentity(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"public_key":"age1ownerkey"}`))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	identity, err := client.GetOwnerIdentity(context.Background())
	if err != nil {
		t.Fatalf("GetOwnerIdentity: %v", err)
	}
	if gotPath != "/auth/identity" || gotAuth != "Bearer token" {
		t.Errorf("request = %s with Authorization %q, want /auth/identity with the bearer token", gotPath, gotAuth)
	}
	if identity.PublicKey == nil || *identity.PublicKey != "age1ownerkey" {
		t.Errorf("PublicKey = %v, want age1ownerkey", identity.PublicKey)
	}
}

// An owner who hasn't completed a first registration has no escrowed key
// yet. The server answers 200 with the field absent, not an error, so the
// caller can tell "nothing to add" from "request failed".
func TestClient_GetOwnerIdentity_NoEscrowedKeyYet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	identity, err := client.GetOwnerIdentity(context.Background())
	if err != nil {
		t.Fatalf("GetOwnerIdentity: %v", err)
	}
	if identity.PublicKey != nil {
		t.Errorf("PublicKey = %q, want nil", *identity.PublicKey)
	}
}

func TestClient_GetOwnerIdentity_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "missing or invalid bearer token or session"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetOwnerIdentity(context.Background())
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("err = %v, want an *APIError with status 401", err)
	}
}

func TestClient_ListObjectsFiltered_TagFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter hushhush.ListObjectsFilter
		want   url.Values
	}{
		{"one tag", hushhush.ListObjectsFilter{Tags: []string{"prod"}}, url.Values{"tag": {"prod"}}},
		{"every tag must match, sent as a repeated parameter", hushhush.ListObjectsFilter{Tags: []string{"prod", "ci"}}, url.Values{"tag": {"prod", "ci"}}},
		{"alongside used_by", hushhush.ListObjectsFilter{UsedBy: "homelab/vps", Tags: []string{"prod"}}, url.Values{"tag": {"prod"}, "used_by": {"homelab/vps"}}},
		{"empty filter sends nothing", hushhush.ListObjectsFilter{}, url.Values{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]hushhush.ObjectMetadata{{Slug: "obj-1"}})
			}))
			defer srv.Close()

			client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			objects, err := client.ListObjectsFiltered(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("ListObjectsFiltered: %v", err)
			}
			if !reflect.DeepEqual(gotQuery, tt.want) {
				t.Errorf("query = %v, want %v", gotQuery, tt.want)
			}
			if len(objects) != 1 || objects[0].Slug != "obj-1" {
				t.Errorf("objects = %+v, want one entry for obj-1", objects)
			}
		})
	}
}

func TestClient_ListObjectsFiltered_RequiresCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "missing or invalid bearer token"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.ListObjectsFiltered(context.Background(), hushhush.ListObjectsFilter{Tags: []string{"prod"}})
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("err = %v, want an *APIError with status 401", err)
	}
}
