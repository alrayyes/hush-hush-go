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
	"net/http"
	"os"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
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

// preferExampleTransport sets Prism's own "Prefer: example=<name>" header
// on every request, selecting one of a spec response's several named
// examples deterministically. Without it, Prism always returns the first
// example it finds regardless of query parameters — GET /consumers's
// "paginated" example never comes back on its own, no matter what filter
// a request sends, so a test that wants that shape has to ask for it by
// name (confirmed by running Prism against the pinned spec locally).
type preferExampleTransport struct {
	example string
}

func (t preferExampleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Prefer", "example="+t.example)
	return http.DefaultTransport.RoundTrip(req)
}

func mustClientPreferringExample(t *testing.T, example string) *hushhush.Client {
	t.Helper()
	baseURL := os.Getenv("HUSH_HUSH_BASE_URL")
	if baseURL == "" {
		t.Fatal("HUSH_HUSH_BASE_URL must point at a running Prism mock (see ci.yml's contract job)")
	}
	client, err := hushhush.NewClient(baseURL,
		hushhush.WithAPIKey("prism-does-not-check-this"),
		hushhush.WithHTTPClient(&http.Client{Transport: preferExampleTransport{example: example}}),
	)
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

func TestContract_AuthStatus(t *testing.T) {
	client := mustClient(t)
	if _, err := client.AuthStatus(context.Background()); err != nil {
		t.Fatalf("AuthStatus: %v", err)
	}
}

func TestContract_CreateGetDeleteObject(t *testing.T) {
	client := mustClient(t)
	ctx := context.Background()

	if _, err := client.CreateObject(ctx, hushhush.CreateObjectRequest{
		Slug:  "contract-test-object",
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

func TestContract_ListConsumers_PlainList(t *testing.T) {
	client := mustClient(t)
	result, err := client.ListConsumers(context.Background(), hushhush.ConsumerFilter{})
	if err != nil {
		t.Fatalf("ListConsumers: %v", err)
	}
	if result.Page != nil {
		t.Errorf("Page = %+v, want nil for an empty filter", result.Page)
	}
	if len(result.Names) == 0 {
		t.Error("Names is empty, want the mock spec's example consumer names")
	}
}

func TestContract_ListConsumers_FilteredPage(t *testing.T) {
	// Selects the spec's "paginated" example explicitly (see
	// preferExampleTransport) — Prism doesn't switch response shape based
	// on query parameters on its own, so a plain q filter alone wouldn't
	// reliably exercise the ConsumersPage decode path.
	client := mustClientPreferringExample(t, "paginated")
	q := "homelab"
	result, err := client.ListConsumers(context.Background(), hushhush.ConsumerFilter{Q: &q})
	if err != nil {
		t.Fatalf("ListConsumers: %v", err)
	}
	if result.Names != nil {
		t.Errorf("Names = %v, want nil for a non-empty filter", result.Names)
	}
	if result.Page == nil || len(result.Page.Consumers) == 0 {
		t.Fatalf("Page = %+v, want at least one consumer entry", result.Page)
	}

	// The mock spec's "paginated" example records one consumer
	// (homelab/mattermost) with no public key at all, and one
	// (homelab/vps-docker) with one registered — covering both the
	// with-key and no-key-registered shapes of ConsumerEntry.PublicKey in
	// one call.
	var sawKey, sawNoKey bool
	for _, entry := range result.Page.Consumers {
		if entry.PublicKey != nil {
			sawKey = true
		} else {
			sawNoKey = true
		}
	}
	if !sawKey {
		t.Error("no consumer in the page has a registered public key")
	}
	if !sawNoKey {
		t.Error("no consumer in the page is missing a public key")
	}
}

func TestContract_GetConsumerPublicKey(t *testing.T) {
	client := mustClientPreferringExample(t, "paginated")
	key, err := client.GetConsumerPublicKey(context.Background(), "homelab/vps-docker")
	if err != nil {
		t.Fatalf("GetConsumerPublicKey: %v", err)
	}
	if key == nil {
		t.Error("key = nil, want the mock spec's registered public key for homelab/vps-docker")
	}
}

func TestContract_AddConsumer(t *testing.T) {
	client := mustClient(t)
	entry, err := client.AddConsumer(context.Background(), hushhush.AddConsumerRequest{Name: "hush-hush-go-contract-test-consumer"})
	if err != nil {
		t.Fatalf("AddConsumer: %v", err)
	}
	if entry.SecretCount != 0 {
		t.Errorf("SecretCount = %d, want 0 for a brand-new consumer", entry.SecretCount)
	}
}

func TestContract_UpdateConsumer_RegistersPublicKey(t *testing.T) {
	client := mustClient(t)
	publicKey := "age1exampleplaceholderpublickey"
	entry, err := client.UpdateConsumer(context.Background(), "homelab/vps-docker", hushhush.UpdateConsumerRequest{PublicKey: &publicKey})
	if err != nil {
		t.Fatalf("UpdateConsumer: %v", err)
	}
	if entry.PublicKey == nil || *entry.PublicKey != publicKey {
		t.Errorf("PublicKey = %v, want %q", entry.PublicKey, publicKey)
	}
}

func TestContract_DeleteConsumer(t *testing.T) {
	client := mustClient(t)
	if err := client.DeleteConsumer(context.Background(), "hush-hush-go-contract-test-consumer"); err != nil {
		t.Fatalf("DeleteConsumer: %v", err)
	}
}
