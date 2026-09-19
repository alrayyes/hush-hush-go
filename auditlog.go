package hushhush

import (
	"context"
	"net/http"
	"time"

	"github.com/alrayyes/hush-hush-go/v2/internal/genclient"
)

// AuditLogFilter narrows a QueryAuditLog call. Filters combine with AND
// when more than one is set. QueryAuditLog returns one page of the
// matching result set, oldest first — set Limit to the page size (server
// default 50, max 500) and, to fetch the next page, After to the
// previous page's last returned entry's own Id; a short page (fewer than
// Limit entries) means nothing is left.
type AuditLogFilter struct {
	ObjectID *string
	Caller   *string
	// Actor restricts to entries authenticated by this verified actor — a
	// token id, or the admin account's own actor id. Unlike Caller, this
	// is never self-reported.
	Actor *string
	From  *time.Time
	To    *time.Time
	// After restricts to entries recorded after this entry id.
	After *int64
	// Limit caps how many entries a single call returns.
	Limit *int32
}

// QueryAuditLog returns one page of the audit log entries matching
// filter, oldest first, exactly as hush-hush returns them. No credential
// is required.
func (c *Client) QueryAuditLog(ctx context.Context, filter AuditLogFilter) ([]AuditLogEntry, error) {
	params := &genclient.QueryAuditLogParams{
		ObjectId: filter.ObjectID,
		Caller:   filter.Caller,
		Actor:    filter.Actor,
		From:     filter.From,
		To:       filter.To,
		After:    filter.After,
		Limit:    filter.Limit,
	}
	resp, err := c.api.QueryAuditLogWithResponse(ctx, params, c.authEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.HTTPResponse.Header, resp.Body)
	}
	return *resp.JSON200, nil
}
