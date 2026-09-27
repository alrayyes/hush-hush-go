package hushhush_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

func TestClient_ListConsumers_PlainList(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]string{"homelab/mattermost", "homelab/vps-docker"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.ListConsumers(context.Background(), hushhush.ConsumerFilter{})
	if err != nil {
		t.Fatalf("ListConsumers: %v", err)
	}
	if result.Page != nil {
		t.Errorf("Page = %+v, want nil for an empty filter", result.Page)
	}
	if want := []string{"homelab/mattermost", "homelab/vps-docker"}; len(result.Names) != len(want) || result.Names[0] != want[0] || result.Names[1] != want[1] {
		t.Errorf("Names = %v, want %v", result.Names, want)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want no query params for an empty filter", gotQuery)
	}
}

func TestClient_ListConsumers_FilteredPage(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumersPage{
			Consumers: []hushhush.ConsumerEntry{
				{Name: "homelab/vps-docker", SecretCount: 1, PublicKey: strPtr("age1exampleplaceholderpublickey")},
			},
			Total: 1,
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	q := "vps"
	result, err := client.ListConsumers(context.Background(), hushhush.ConsumerFilter{Q: &q})
	if err != nil {
		t.Fatalf("ListConsumers: %v", err)
	}
	if result.Names != nil {
		t.Errorf("Names = %v, want nil for a non-empty filter", result.Names)
	}
	if result.Page == nil {
		t.Fatal("Page = nil, want a page for a non-empty filter")
	}
	if result.Page.Total != 1 || len(result.Page.Consumers) != 1 || result.Page.Consumers[0].Name != "homelab/vps-docker" {
		t.Errorf("Page = %+v, want one entry for homelab/vps-docker", result.Page)
	}
	if result.Page.Consumers[0].PublicKey == nil || *result.Page.Consumers[0].PublicKey != "age1exampleplaceholderpublickey" {
		t.Errorf("PublicKey = %v, want the registered key", result.Page.Consumers[0].PublicKey)
	}
	if gotQuery != "q=vps" {
		t.Errorf("query = %q, want %q", gotQuery, "q=vps")
	}
}

func TestClient_GetConsumerPublicKey_Registered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumersPage{
			Consumers: []hushhush.ConsumerEntry{
				{Name: "homelab/vps-docker", SecretCount: 1, PublicKey: strPtr("age1exampleplaceholderpublickey")},
			},
			Total: 1,
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	key, err := client.GetConsumerPublicKey(context.Background(), "homelab/vps-docker")
	if err != nil {
		t.Fatalf("GetConsumerPublicKey: %v", err)
	}
	if key == nil || *key != "age1exampleplaceholderpublickey" {
		t.Errorf("key = %v, want the registered key", key)
	}
}

func TestClient_GetConsumerPublicKey_NoKeyRegistered(t *testing.T) {
	// homelab/mattermost is a recorded consumer with no public_key field
	// at all — absence reported as a nil *string, never an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumersPage{
			Consumers: []hushhush.ConsumerEntry{
				{Name: "homelab/mattermost", SecretCount: 2},
			},
			Total: 1,
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	key, err := client.GetConsumerPublicKey(context.Background(), "homelab/mattermost")
	if err != nil {
		t.Fatalf("GetConsumerPublicKey: %v", err)
	}
	if key != nil {
		t.Errorf("key = %q, want nil for a consumer with no registered key", *key)
	}
}

func TestClient_GetConsumerPublicKey_UnknownConsumer(t *testing.T) {
	// The directory has no entry at all for this name — still reported
	// as a nil *string, not an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumersPage{Consumers: []hushhush.ConsumerEntry{}, Total: 0})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	key, err := client.GetConsumerPublicKey(context.Background(), "no-such-consumer")
	if err != nil {
		t.Fatalf("GetConsumerPublicKey: %v", err)
	}
	if key != nil {
		t.Errorf("key = %q, want nil for an unknown consumer", *key)
	}
}

func TestClient_AddConsumer(t *testing.T) {
	var gotBody hushhush.AddConsumerRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumerEntry{Name: "homelab/new-device", SecretCount: 0})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	entry, err := client.AddConsumer(context.Background(), hushhush.AddConsumerRequest{Name: "homelab/new-device"})
	if err != nil {
		t.Fatalf("AddConsumer: %v", err)
	}
	if entry.Name != "homelab/new-device" || entry.SecretCount != 0 {
		t.Errorf("entry = %+v, want a fresh entry with SecretCount 0", entry)
	}
	if gotBody.Name != "homelab/new-device" {
		t.Errorf("request body name = %q, want %q", gotBody.Name, "homelab/new-device")
	}
}

func TestClient_AddConsumer_AlreadyExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "consumer already exists"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.AddConsumer(context.Background(), hushhush.AddConsumerRequest{Name: "homelab/vps-docker"})
	if err == nil {
		t.Fatal("AddConsumer with a duplicate name: want error, got nil")
	}
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *hushhush.APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusConflict)
	}
}

func TestClient_UpdateConsumer_RegistersPublicKey(t *testing.T) {
	var gotBody hushhush.UpdateConsumerRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/consumers/homelab/vps-docker" {
			t.Errorf("path = %q, want /consumers/homelab/vps-docker", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumerEntry{
			Name:        "homelab/vps-docker",
			SecretCount: 3,
			PublicKey:   strPtr("age1exampleplaceholderpublickey"),
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	publicKey := "age1exampleplaceholderpublickey"
	entry, err := client.UpdateConsumer(context.Background(), "homelab/vps-docker", hushhush.UpdateConsumerRequest{PublicKey: &publicKey})
	if err != nil {
		t.Fatalf("UpdateConsumer: %v", err)
	}
	if entry.PublicKey == nil || *entry.PublicKey != publicKey {
		t.Errorf("PublicKey = %v, want %q", entry.PublicKey, publicKey)
	}
	if gotBody.PublicKey == nil || *gotBody.PublicKey != publicKey {
		t.Errorf("request body PublicKey = %v, want %q", gotBody.PublicKey, publicKey)
	}
	if gotBody.Name != nil {
		t.Errorf("request body Name = %v, want nil (no rename requested)", gotBody.Name)
	}
}

func TestClient_UpdateConsumer_UnknownName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "unknown consumer"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	newName := "homelab/renamed"
	_, err = client.UpdateConsumer(context.Background(), "no-such-consumer", hushhush.UpdateConsumerRequest{Name: &newName})
	if err == nil {
		t.Fatal("UpdateConsumer on an unknown name: want error, got nil")
	}
}

func TestClient_DeleteConsumer(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.DeleteConsumer(context.Background(), "homelab/vps-docker"); err != nil {
		t.Fatalf("DeleteConsumer: %v", err)
	}
	if gotPath != "/consumers/homelab/vps-docker" {
		t.Errorf("path = %q, want /consumers/homelab/vps-docker", gotPath)
	}
}

func TestClient_DeleteConsumer_UnknownName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "unknown consumer"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.DeleteConsumer(context.Background(), "no-such-consumer"); err == nil {
		t.Fatal("DeleteConsumer on an unknown name: want error, got nil")
	}
}

func strPtr(s string) *string { return &s }
