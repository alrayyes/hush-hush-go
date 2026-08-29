package hushhush

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alrayyes/hush-hush-go/internal/genclient"
)

const apiKeyEnvVar = "HUSH_HUSH_API_KEY" //nolint:gosec // this is an env var *name*, not a credential value

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

// Client is the hush-hush SDK's entry point. Construct one with NewClient.
type Client struct {
	api    *genclient.ClientWithResponses
	apiKey string
}

// Option configures a Client constructed by NewClient.
type Option func(*config)

type config struct {
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

// WithAPIKey sets the bearer credential used on write operations
// (create/update/delete). If not set, NewClient falls back to the
// HUSH_HUSH_API_KEY environment variable.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithHTTPClient overrides the underlying *http.Client. Its Transport, if
// set, is wrapped with hush-hush's retry behavior rather than replaced.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) { c.httpClient = hc }
}

// WithTimeout sets the per-request timeout. Defaults to 30 seconds.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithMaxRetries sets how many times a request is retried after network
// failure or a 5xx/429 response. Defaults to 3.
func WithMaxRetries(n int) Option {
	return func(c *config) { c.maxRetries = n }
}

// NewClient constructs a Client for the hush-hush instance at baseURL.
func NewClient(baseURL string, opts ...Option) (*Client, error) {
	cfg := &config{
		apiKey:     os.Getenv(apiKeyEnvVar),
		httpClient: &http.Client{},
		timeout:    defaultTimeout,
		maxRetries: defaultMaxRetries,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	base := cfg.httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	hc := &http.Client{
		Transport:     &retryTransport{base: base, maxRetries: cfg.maxRetries},
		Timeout:       cfg.timeout,
		CheckRedirect: cfg.httpClient.CheckRedirect,
		Jar:           cfg.httpClient.Jar,
	}

	api, err := genclient.NewClientWithResponses(baseURL, genclient.WithHTTPClient(hc))
	if err != nil {
		return nil, fmt.Errorf("hushhush: %w", err)
	}
	return &Client{api: api, apiKey: cfg.apiKey}, nil
}

func (c *Client) authEditor(_ context.Context, req *http.Request) error {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	return nil
}

// Health reports whether hush-hush is up.
func (c *Client) Health(ctx context.Context) (*Health, error) {
	resp, err := c.api.HealthWithResponse(ctx, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateObject stores an already-sealed value under a new object id.
// Requires a credential (see WithAPIKey/HUSH_HUSH_API_KEY). caller, if
// non-empty, is recorded in the audit log as the X-Caller header.
func (c *Client) CreateObject(ctx context.Context, req CreateObjectRequest, caller string) (*ObjectMetadata, error) {
	params := &genclient.CreateObjectParams{}
	if caller != "" {
		params.XCaller = &caller
	}
	resp, err := c.api.CreateObjectWithResponse(ctx, params, req, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON201, nil
}

// GetObject fetches an object's sealed ciphertext exactly as stored — this
// SDK never decrypts it, the same as the server. No credential is required.
func (c *Client) GetObject(ctx context.Context, id string, caller string) ([]byte, error) {
	params := &genclient.GetObjectParams{}
	if caller != "" {
		params.XCaller = &caller
	}
	resp, err := c.api.GetObjectWithResponse(ctx, id, params, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.Body, nil
}

// UpdateObject replaces the stored ciphertext for an existing object. The
// object's id and used-by metadata are unchanged. Requires a credential.
func (c *Client) UpdateObject(ctx context.Context, id string, req UpdateObjectRequest, caller string) (*ObjectMetadata, error) {
	params := &genclient.UpdateObjectParams{}
	if caller != "" {
		params.XCaller = &caller
	}
	resp, err := c.api.UpdateObjectWithResponse(ctx, id, params, req, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteObject permanently removes an object. Requires a credential.
func (c *Client) DeleteObject(ctx context.Context, id string, caller string) error {
	params := &genclient.DeleteObjectParams{}
	if caller != "" {
		params.XCaller = &caller
	}
	resp, err := c.api.DeleteObjectWithResponse(ctx, id, params, c.authEditor)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return nil
}

// GetObjectUsedBy returns the recorded list of consumers for an object. No
// credential is required.
func (c *Client) GetObjectUsedBy(ctx context.Context, id string) (*UsedBy, error) {
	resp, err := c.api.GetObjectUsedByWithResponse(ctx, id, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}
