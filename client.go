package hushhush

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alrayyes/hush-hush-go/v4/internal/genclient"
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

// AuthStatus reports whether hush-hush has an admin account bootstrapped
// yet. Unauthenticated - no credential is required to call it.
func (c *Client) AuthStatus(ctx context.Context) (*AuthStatus, error) {
	resp, err := c.api.GetAuthStatusWithResponse(ctx, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// GetOwnerIdentity returns the owner's escrowed identity public key, the age
// recipient a client adds to its own recipient list before sealing when it
// wants CreateObjectRequest.KeepReadableCopy honored: the server never adds
// a recipient itself, so without this key the flag can't be acted on. Needs a
// credential (an API key works, as of hush-hush v2.54.0).
//
// An owner who hasn't completed a first registration has no escrowed key yet.
// That isn't an error: the call succeeds with PublicKey nil, and there is
// nothing to add.
func (c *Client) GetOwnerIdentity(ctx context.Context) (*OwnerIdentity, error) {
	resp, err := c.api.GetOwnerIdentityWithResponse(ctx, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// ObjectOption adjusts one call that addresses an existing object by name:
// GetObject, UpdateObject, DeleteObject and GetObjectUsedBy.
type ObjectOption func(*objectConfig)

type objectConfig struct {
	variantID string
	consumer  string
}

// WithVariantID addresses one variant of a name that has several. A name can
// hold a different value per consumer, each under its own UUID, which
// ObjectMetadata.Id returns. Without it a name with one variant is addressed
// as always, and a name with several is a 409 *APIError. An id that doesn't
// belong to the name is a 404. A malformed id is refused before any request
// is sent.
func WithVariantID(id string) ObjectOption {
	return func(c *objectConfig) { c.variantID = id }
}

// WithConsumer addresses the one variant of a name whose consumers include
// consumer, for a caller that knows who reads the value and not its UUID. A
// consumer that no variant names is a 404 *APIError. Combining it with
// WithVariantID is the server's 400, returned as an *APIError with its
// message. Without either, a name with one variant is addressed as always.
func WithConsumer(consumer string) ObjectOption {
	return func(c *objectConfig) { c.consumer = consumer }
}

// variantSelector is the generated client's two ways to pick a variant, each
// nil when not given.
type variantSelector struct {
	id       *genclient.ObjectId
	consumer *genclient.ObjectConsumer
}

// variantParams turns the options into the generated client's id and
// consumer parameters.
func variantParams(opts []ObjectOption) (variantSelector, error) {
	var cfg objectConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var sel variantSelector
	if cfg.consumer != "" {
		sel.consumer = &cfg.consumer
	}
	if cfg.variantID != "" {
		var id genclient.ObjectId
		if err := id.UnmarshalText([]byte(cfg.variantID)); err != nil {
			return variantSelector{}, fmt.Errorf("hushhush: variant id %q is not a UUID: %w", cfg.variantID, err)
		}
		sel.id = &id
	}

	return sel, nil
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
func (c *Client) GetObject(ctx context.Context, id string, caller string, opts ...ObjectOption) ([]byte, error) {
	sel, err := variantParams(opts)
	if err != nil {
		return nil, err
	}
	params := &genclient.GetObjectParams{Id: sel.id, Consumer: sel.consumer}
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
func (c *Client) UpdateObject(ctx context.Context, id string, req UpdateObjectRequest, caller string, opts ...ObjectOption) (*ObjectMetadata, error) {
	sel, err := variantParams(opts)
	if err != nil {
		return nil, err
	}
	params := &genclient.UpdateObjectParams{Id: sel.id, Consumer: sel.consumer}
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
func (c *Client) DeleteObject(ctx context.Context, id string, caller string, opts ...ObjectOption) error {
	sel, err := variantParams(opts)
	if err != nil {
		return err
	}
	params := &genclient.DeleteObjectParams{Id: sel.id, Consumer: sel.consumer}
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
func (c *Client) GetObjectUsedBy(ctx context.Context, id string, opts ...ObjectOption) (*UsedBy, error) {
	sel, err := variantParams(opts)
	if err != nil {
		return nil, err
	}
	resp, err := c.api.GetObjectUsedByWithResponse(ctx, id, &genclient.GetObjectUsedByParams{Id: sel.id, Consumer: sel.consumer}, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return resp.JSON200, nil
}

// ListObjectsFilter narrows a ListObjectsFiltered call. The zero value asks
// for every object, the same as ListObjects with an empty usedBy.
type ListObjectsFilter struct {
	// UsedBy restricts to objects whose recorded used_by lineage includes
	// this consumer. "" means no filter.
	UsedBy string
	// Tags restricts to objects carrying every one of these tags (all must
	// match, case-insensitive). Empty means no filter.
	Tags []string
}

// ListObjects returns every stored object's metadata (id, used_by,
// description — never the sealed value), optionally narrowed to objects
// whose recorded used_by lineage includes usedBy ("" means no filter).
// Unlike every other read in this SDK, this requires a credential: an
// id-scoped read only discloses what the caller already knows the id of,
// but enumerating every object is a capability none of those grant on
// their own.
//
// The list is read a page of 500 at a time and returned whole, in the
// server's order, so it stays complete however many objects there are. A
// failed page returns its error and no rows.
//
// To filter by tag as well, use ListObjectsFiltered.
func (c *Client) ListObjects(ctx context.Context, usedBy string) ([]ObjectMetadata, error) {
	return c.ListObjectsFiltered(ctx, ListObjectsFilter{UsedBy: usedBy})
}

// ListObjectsFiltered is ListObjects with the full set of GET /objects
// filters: used_by and tag. It requires a credential for the same reason.
// Tags are sent as a repeated query parameter and an object must carry every
// one, so listing by [prod] returns the prod objects and listing by
// [prod, ci] only those carrying both.
func (c *Client) ListObjectsFiltered(ctx context.Context, filter ListObjectsFilter) ([]ObjectMetadata, error) {
	return readAllPages(ctx, func(ctx context.Context, limit, offset int32) (page[ObjectMetadata], error) {
		params := &genclient.ListObjectsParams{Limit: &limit, Offset: &offset}
		if filter.UsedBy != "" {
			params.UsedBy = &filter.UsedBy
		}
		if len(filter.Tags) > 0 {
			params.Tag = &filter.Tags
		}
		resp, err := c.api.ListObjectsWithResponse(ctx, params, c.authEditor)
		if err != nil {
			return page[ObjectMetadata]{}, err
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return page[ObjectMetadata]{}, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
		}
		total, hasTotal := totalCount(resp.HTTPResponse.Header)
		return page[ObjectMetadata]{rows: *resp.JSON200, total: total, hasTotal: hasTotal}, nil
	})
}
