package hushhush

import (
	"context"
	"net/http"

	"github.com/alrayyes/hush-hush-go/v4/internal/genclient"
)

// CreateConsumerToken issues a new read token scoped to req.Consumer. The
// returned value's Value field holds the raw token — it is never
// recoverable again once this call returns. Requires a credential.
func (c *Client) CreateConsumerToken(ctx context.Context, req CreateConsumerTokenRequest) (*ConsumerTokenWithValue, error) {
	resp, err := c.api.CreateConsumerTokenWithResponse(ctx, &genclient.CreateConsumerTokenParams{}, req, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON201, nil
}

// ListConsumerTokens returns every issued consumer token's metadata —
// never a raw value, which by design no longer exists anywhere to return
// once a token is created. Requires a credential.
func (c *Client) ListConsumerTokens(ctx context.Context) ([]ConsumerTokenMetadata, error) {
	resp, err := c.api.ListConsumerTokensWithResponse(ctx, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return *resp.JSON200, nil
}

// RotateConsumerToken replaces the secret and expiry of the consumer token
// issued under id, keeping its id, consumer, and description unchanged.
// The old secret stops authenticating immediately, and the returned
// value's Value field holds the new raw token — the same as
// CreateConsumerToken, it's shown here only. Unlike RevokeConsumerToken,
// an id that's unknown, already revoked, or already expired is rejected
// with an *APIError wrapping a 400 or 404. Requires a credential.
func (c *Client) RotateConsumerToken(ctx context.Context, id string, req RotateConsumerTokenRequest) (*ConsumerTokenWithValue, error) {
	resp, err := c.api.RotateConsumerTokenWithResponse(ctx, id, &genclient.RotateConsumerTokenParams{}, req, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// RevokeConsumerToken invalidates the consumer token issued under id.
// Revoking an id that's already expired or doesn't exist isn't an error —
// both leave every other token's state unaffected either way. Requires a
// credential.
func (c *Client) RevokeConsumerToken(ctx context.Context, id string) error {
	resp, err := c.api.RevokeConsumerTokenWithResponse(ctx, id, &genclient.RevokeConsumerTokenParams{}, c.authEditor)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return nil
}

// PurgeConsumerToken permanently removes the consumer token issued under
// id, once it's already revoked or past its expiry — RevokeConsumerToken's
// soft-delete stays the only way to invalidate a still-active token.
// Purging a still-active token is rejected with an *APIError wrapping a
// 409. Requires a credential.
func (c *Client) PurgeConsumerToken(ctx context.Context, id string) error {
	resp, err := c.api.PurgeConsumerTokenWithResponse(ctx, id, &genclient.PurgeConsumerTokenParams{}, c.authEditor)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return nil
}
