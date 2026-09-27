package hushhush

import (
	"context"
	"net/http"

	"github.com/alrayyes/hush-hush-go/v4/internal/genclient"
)

// ConsumerFilter narrows a ListConsumers call. The zero value (every field
// nil) asks for the plain, unpaginated list of consumer names — GET
// /consumers's default response shape. Setting any field switches the
// response to the paginated ConsumersPage shape instead: there's no way to
// ask for a filtered plain list, or an unfiltered page — the underlying
// API doesn't offer one.
type ConsumerFilter struct {
	// Q restricts to consumers whose name contains this substring,
	// case-insensitive.
	Q *string
	// Page is the 1-based page number.
	Page *int32
	// PageSize caps how many consumers a single page returns (server
	// default 20, max 100).
	PageSize *int32
}

func (f ConsumerFilter) empty() bool {
	return f.Q == nil && f.Page == nil && f.PageSize == nil
}

// ConsumersResult is ListConsumers's return value. Exactly one field is
// populated, matching whichever of GET /consumers's two response shapes
// the server sent back for that call's filter.
type ConsumersResult struct {
	// Names holds the plain, sorted list of every consumer name — set
	// only when the ListConsumers call used an empty ConsumerFilter.
	Names []string
	// Page holds one page of ConsumerEntry plus the total matching
	// count — set only when the ListConsumers call's filter had any
	// field set.
	Page *ConsumersPage
}

// ListConsumers returns hush-hush's consumer directory. Requires a
// credential, the same as ListObjects — listing needs no id the caller
// already holds.
//
// Called with an empty ConsumerFilter, the result's Names holds the
// plain, sorted list of every consumer name and Page is nil. Called with
// any filter field set, the result's Page holds one page of ConsumerEntry
// (name, secret count, and public key when registered) plus the total
// matching count, and Names is nil. This mirrors GET /consumers's own
// union response shape rather than flattening it into one — a caller
// that only wants one consumer's public key by exact name should use
// GetConsumerPublicKey instead of picking the union apart itself.
func (c *Client) ListConsumers(ctx context.Context, filter ConsumerFilter) (*ConsumersResult, error) {
	params := &genclient.ListConsumersParams{
		Q:        filter.Q,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	resp, err := c.api.ListConsumersWithResponse(ctx, params, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}

	if filter.empty() {
		names, err := resp.JSON200.AsListConsumers200JSONResponseBody0()
		if err != nil {
			return nil, err
		}
		return &ConsumersResult{Names: names}, nil
	}
	page, err := resp.JSON200.AsConsumersPage()
	if err != nil {
		return nil, err
	}
	return &ConsumersResult{Page: &page}, nil
}

// GetConsumerPublicKey resolves one consumer's registered age public key
// by exact name, without requiring the caller to pick apart
// ListConsumers's union response itself — the use case
// alrayyes/hush-hush-cli#125 needs. Built on ListConsumers with Q set to
// name, then matched exactly against the returned page (Q itself is a
// substring match, so a shorter name could otherwise match more than one
// entry).
//
// Returns nil, nil — no error — both when name isn't in the directory at
// all and when it is but has no key registered: either way, there's
// simply no public key to hand back. Requires a credential, the same as
// ListConsumers.
func (c *Client) GetConsumerPublicKey(ctx context.Context, name string) (*string, error) {
	result, err := c.ListConsumers(ctx, ConsumerFilter{Q: &name})
	if err != nil {
		return nil, err
	}
	if result.Page == nil {
		return nil, nil
	}
	for _, entry := range result.Page.Consumers {
		if entry.Name == name {
			return entry.PublicKey, nil
		}
	}
	return nil, nil
}

// AddConsumer adds req.Name to the directory with no secret referencing
// it yet — the returned entry's SecretCount is 0 until some object's
// used_by list actually references it. Requires a credential. Rejected
// with an *APIError wrapping a 409 if the name already appears in the
// directory.
func (c *Client) AddConsumer(ctx context.Context, req AddConsumerRequest) (*ConsumerEntry, error) {
	resp, err := c.api.AddConsumerWithResponse(ctx, &genclient.AddConsumerParams{}, req, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON201, nil
}

// UpdateConsumer renames a consumer and/or registers its age public key.
// req.Name, if set, replaces name in every stored object's used_by list
// that currently records it — if the target name already has its own
// recorded objects, they merge under it. req.PublicKey, if set, registers
// or replaces that key on the resulting name, upserting a directory entry
// for it even if it had none before. Sending both fields renames first,
// then sets the key on the resulting name. Requires a credential.
// Renaming an unknown name is rejected with an *APIError wrapping a 404;
// registering a public key alone never is.
func (c *Client) UpdateConsumer(ctx context.Context, name string, req UpdateConsumerRequest) (*ConsumerEntry, error) {
	resp, err := c.api.UpdateConsumerWithResponse(ctx, name, &genclient.UpdateConsumerParams{}, req, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteConsumer strips name from the used_by list of every stored object
// that currently records it. The objects themselves aren't touched
// otherwise, and none are deleted even if this empties their used_by
// list. Requires a credential.
func (c *Client) DeleteConsumer(ctx context.Context, name string) error {
	resp, err := c.api.DeleteConsumerWithResponse(ctx, name, &genclient.DeleteConsumerParams{}, c.authEditor)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return nil
}
