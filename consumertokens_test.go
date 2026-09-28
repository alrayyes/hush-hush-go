package hushhush_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

func TestClient_CreateConsumerToken(t *testing.T) {
	var gotBody hushhush.CreateConsumerTokenRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumerTokenWithValue{
			Id:          "tok_1",
			Consumer:    "homelab/vps-docker",
			Description: "deploy read token",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
			Value:       "raw-token-value",
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tok, err := client.CreateConsumerToken(context.Background(), hushhush.CreateConsumerTokenRequest{
		Consumer:    "homelab/vps-docker",
		Description: "deploy read token",
		TtlSeconds:  7776000,
	})
	if err != nil {
		t.Fatalf("CreateConsumerToken: %v", err)
	}
	if tok.Value != "raw-token-value" {
		t.Errorf("Value = %q, want %q", tok.Value, "raw-token-value")
	}
	if gotBody.Consumer != "homelab/vps-docker" || gotBody.TtlSeconds != 7776000 {
		t.Errorf("request body = %+v, want consumer homelab/vps-docker with ttl 7776000", gotBody)
	}
}

func TestClient_ListConsumerTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]hushhush.ConsumerTokenMetadata{
			{Id: "tok_1", Consumer: "homelab/vps-docker", Description: "deploy read token"},
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tokens, err := client.ListConsumerTokens(context.Background())
	if err != nil {
		t.Fatalf("ListConsumerTokens: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Id != "tok_1" {
		t.Errorf("tokens = %+v, want one entry with id tok_1", tokens)
	}
}

func TestClient_RotateConsumerToken(t *testing.T) {
	var gotPath string
	var gotBody hushhush.RotateConsumerTokenRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hushhush.ConsumerTokenWithValue{
			Id:          "tok_1",
			Consumer:    "homelab/vps-docker",
			Description: "deploy read token",
			Value:       "new-raw-token-value",
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tok, err := client.RotateConsumerToken(context.Background(), "tok_1", hushhush.RotateConsumerTokenRequest{TtlSeconds: 7776000})
	if err != nil {
		t.Fatalf("RotateConsumerToken: %v", err)
	}
	if tok.Value != "new-raw-token-value" {
		t.Errorf("Value = %q, want %q", tok.Value, "new-raw-token-value")
	}
	if gotPath != "/consumer-tokens/tok_1/rotate" {
		t.Errorf("path = %q, want /consumer-tokens/tok_1/rotate", gotPath)
	}
	if gotBody.TtlSeconds != 7776000 {
		t.Errorf("request body TtlSeconds = %d, want 7776000", gotBody.TtlSeconds)
	}
}

func TestClient_RotateConsumerToken_UnknownID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "unknown consumer token"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.RotateConsumerToken(context.Background(), "no-such-id", hushhush.RotateConsumerTokenRequest{TtlSeconds: 60})
	if err == nil {
		t.Fatal("RotateConsumerToken on an unknown id: want error, got nil")
	}
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *hushhush.APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
}

func TestClient_RevokeConsumerToken(t *testing.T) {
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

	if err := client.RevokeConsumerToken(context.Background(), "tok_1"); err != nil {
		t.Fatalf("RevokeConsumerToken: %v", err)
	}
	if gotPath != "/consumer-tokens/tok_1" {
		t.Errorf("path = %q, want /consumer-tokens/tok_1", gotPath)
	}
}

func TestClient_RevokeConsumerToken_AlreadyInvalid(t *testing.T) {
	// Revoking an id that's already expired or doesn't exist isn't an
	// error — the server reports 204 either way.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.RevokeConsumerToken(context.Background(), "no-such-id"); err != nil {
		t.Fatalf("RevokeConsumerToken: %v", err)
	}
}

func TestClient_PurgeConsumerToken(t *testing.T) {
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

	if err := client.PurgeConsumerToken(context.Background(), "tok_1"); err != nil {
		t.Fatalf("PurgeConsumerToken: %v", err)
	}
	if gotPath != "/consumer-tokens/tok_1/purge" {
		t.Errorf("path = %q, want /consumer-tokens/tok_1/purge", gotPath)
	}
}

func TestClient_PurgeConsumerToken_StillActive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(hushhush.Error{Error: "token is neither revoked nor expired"})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.PurgeConsumerToken(context.Background(), "tok_1")
	if err == nil {
		t.Fatal("PurgeConsumerToken on a still-active token: want error, got nil")
	}
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *hushhush.APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusConflict)
	}
}
