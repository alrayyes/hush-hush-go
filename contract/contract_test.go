//go:build contract

// Package contract runs the client against a Prism mock server generated
// from hush-hush's own pinned spec (see .github/workflows/ci.yml's
// "contract" job) — never a hand-rolled stub. See design.md's testing
// layers: this proves the client's requests/responses conform to the spec;
// it says nothing about whether the real server still matches that spec,
// which is what the Pact consumer contract in ../pact is for.
package contract

import (
	"context"
	"os"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func mustClient(t *testing.T) *hushhush.Client {
	t.Helper()
	baseURL := os.Getenv("HUSH_HUSH_BASE_URL")
	if baseURL == "" {
		t.Fatal("HUSH_HUSH_BASE_URL must point at a running Prism mock (see ci.yml's contract job)")
	}
	client, err := hushhush.NewClient(baseURL, hushhush.WithAPIKey("prism-does-not-check-this"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestContract_Health(t *testing.T) {
	client := mustClient(t)
	if _, err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestContract_CreateGetDeleteObject(t *testing.T) {
	client := mustClient(t)
	ctx := context.Background()

	if _, err := client.CreateObject(ctx, hushhush.CreateObjectRequest{
		Id:    "contract-test-object",
		Value: []byte("sealed-value"),
	}, "hush-hush-go-contract-test"); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	if _, err := client.GetObject(ctx, "contract-test-object", ""); err != nil {
		t.Fatalf("GetObject: %v", err)
	}

	if err := client.DeleteObject(ctx, "contract-test-object", ""); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
}

func TestContract_QueryAuditLog(t *testing.T) {
	client := mustClient(t)
	if _, err := client.QueryAuditLog(context.Background(), hushhush.AuditLogFilter{}); err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
}
