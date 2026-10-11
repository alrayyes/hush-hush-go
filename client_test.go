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
	// The list is read a page at a time, so the paging parameters ride along.
	if want := "used_by=homelab%2Fvps-docker&limit=500&offset=0"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
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
		{"empty filter sends no filter", hushhush.ListObjectsFilter{}, url.Values{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				// The paging parameters are checked in list_pages_test.go.
				gotQuery.Del("limit")
				gotQuery.Del("offset")
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

const variantA = "0b8e1c1e-3f5a-4d0b-8a1f-2c9d7e6b5a41"

// A name can hold a different value per consumer, each under its own UUID
// (hush-hush ADR 33). The wrappers take the variant id as an option, so a
// caller that passes none sends exactly what it always did.
func TestClient_VariantID_IsSentOnEveryObjectCall(t *testing.T) {
	var gotQueries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.URL.Query().Get("id"))
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(hushhush.ObjectMetadata{Slug: "release_token"})
		case r.URL.Path == "/objects/release_token/used-by":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(hushhush.UsedBy{})
		default:
			_, _ = w.Write([]byte("sealed"))
		}
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("k"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()
	withID := hushhush.WithVariantID(variantA)

	if _, err := client.GetObject(ctx, "release_token", "", withID); err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if _, err := client.UpdateObject(ctx, "release_token", hushhush.UpdateObjectRequest{Value: []byte("v")}, "", withID); err != nil {
		t.Fatalf("UpdateObject: %v", err)
	}
	if err := client.DeleteObject(ctx, "release_token", "", withID); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	if _, err := client.GetObjectUsedBy(ctx, "release_token", withID); err != nil {
		t.Fatalf("GetObjectUsedBy: %v", err)
	}

	for i, got := range gotQueries {
		if got != variantA {
			t.Errorf("call %d sent id = %q, want %q", i, got, variantA)
		}
	}
	if len(gotQueries) != 4 {
		t.Errorf("server saw %d calls, want 4", len(gotQueries))
	}
}

func TestClient_VariantID_IsLeftOutWhenNotGiven(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		_, _ = w.Write([]byte("sealed"))
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.GetObject(context.Background(), "release_token", ""); err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if gotRawQuery != "" {
		t.Errorf("query = %q, want none", gotRawQuery)
	}
}

func TestClient_VariantID_ServerRefusalsComeBackAsAPIErrors(t *testing.T) {
	for _, status := range []int{http.StatusConflict, http.StatusNotFound} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"name has several variants; pass id to say which"}`))
		}))

		client, err := hushhush.NewClient(srv.URL)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		_, err = client.GetObject(context.Background(), "release_token", "", hushhush.WithVariantID(variantA))

		var apiErr *hushhush.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
			t.Errorf("status %d: err = %v, want an *APIError with that status", status, err)
		} else if apiErr.Message == "" {
			t.Errorf("status %d: the server's message was dropped", status)
		}
		srv.Close()
	}
}

func TestClient_VariantID_MalformedIsRefusedBeforeAnyRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.GetObject(context.Background(), "release_token", "", hushhush.WithVariantID("not-a-uuid")); err == nil {
		t.Error("GetObject with a malformed variant id returned no error")
	}
	if called {
		t.Error("a malformed variant id still reached the server")
	}
}
