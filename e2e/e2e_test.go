//go:build e2e

// Package e2e is the thin, deliberately sparse layer against a real staging
// hush-hush instance — see design.md's testing layers and
// .github/workflows/e2e.yml, which gates this to release/nightly, never the
// default PR pipeline.
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func TestSmoke_CreateGetDeleteRoundTrip(t *testing.T) {
	baseURL := os.Getenv("HUSH_HUSH_BASE_URL")
	apiKey := os.Getenv("HUSH_HUSH_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skip("HUSH_HUSH_BASE_URL and HUSH_HUSH_API_KEY must be set to run against staging")
	}

	client, err := hushhush.NewClient(baseURL, hushhush.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id := "hush-hush-go-e2e-smoke"
	caller := "hush-hush-go-e2e"

	if _, err := client.CreateObject(ctx, hushhush.CreateObjectRequest{
		Id:    id,
		Value: []byte("e2e-smoke-value"),
	}, caller); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	t.Cleanup(func() {
		_ = client.DeleteObject(context.Background(), id, caller)
	})

	if got, err := client.GetObject(ctx, id, caller); err != nil {
		t.Fatalf("GetObject: %v", err)
	} else if string(got) != "e2e-smoke-value" {
		t.Errorf("GetObject = %q, want %q", got, "e2e-smoke-value")
	}

	if err := client.DeleteObject(ctx, id, caller); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
}
