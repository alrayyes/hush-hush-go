//go:build pact

// Package pact records this SDK's real interactions as a Pact consumer
// contract. Provider verification against hush-hush's actual server has to
// run in hush-hush's own CI (see design.md's Risks — an external dependency
// this repo can't wire up unilaterally); this package's job is only to keep
// producing an up-to-date pact file for that to consume.
package pact

import (
	"context"
	"fmt"
	"testing"

	"github.com/pact-foundation/pact-go/v2/consumer"
	"github.com/pact-foundation/pact-go/v2/matchers"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func newMockProvider(t *testing.T) *consumer.V2HTTPMockProvider {
	t.Helper()
	provider, err := consumer.NewV2Pact(consumer.MockHTTPProviderConfig{
		Consumer: "hush-hush-go",
		Provider: "hush-hush",
		PactDir:  "./pacts",
	})
	if err != nil {
		t.Fatalf("NewV2Pact: %v", err)
	}
	return provider
}

func TestPact_GetObject(t *testing.T) {
	provider := newMockProvider(t)

	err := provider.
		AddInteraction().
		Given("an object exists with id my-object").
		UponReceiving("a request to get an object").
		WithRequest("GET", "/objects/my-object").
		WillRespondWith(200, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/octet-stream"))
			b.BinaryBody([]byte("sealed-bytes"))
		}).
		ExecuteTest(t, func(cfg consumer.MockServerConfig) error {
			client, err := hushhush.NewClient(fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port))
			if err != nil {
				return err
			}
			got, err := client.GetObject(context.Background(), "my-object", "")
			if err != nil {
				return err
			}
			if string(got) != "sealed-bytes" {
				return fmt.Errorf("GetObject = %q, want %q", got, "sealed-bytes")
			}
			return nil
		})
	if err != nil {
		t.Fatalf("pact verification: %v", err)
	}
}

func TestPact_QueryAuditLog(t *testing.T) {
	provider := newMockProvider(t)

	err := provider.
		AddInteraction().
		Given("the audit log has at least one entry").
		UponReceiving("a request to query the audit log").
		WithRequest("GET", "/audit-log").
		WillRespondWith(200, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/json"))
			b.JSONBody(matchers.EachLike(map[string]interface{}{
				"object_id": matchers.Like("my-object"),
				"action":    matchers.Term("read", "create|read|update|delete"),
				"timestamp": matchers.Timestamp(),
			}, 1))
		}).
		ExecuteTest(t, func(cfg consumer.MockServerConfig) error {
			client, err := hushhush.NewClient(fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port))
			if err != nil {
				return err
			}
			entries, err := client.QueryAuditLog(context.Background(), hushhush.AuditLogFilter{})
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				return fmt.Errorf("QueryAuditLog returned no entries")
			}
			return nil
		})
	if err != nil {
		t.Fatalf("pact verification: %v", err)
	}
}
