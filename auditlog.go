package hushhush

import (
	"context"
	"net/http"
	"time"

	"github.com/alrayyes/hush-hush-go/internal/genclient"
)

// AuditLogFilter narrows a QueryAuditLog call. Filters combine with AND
// when more than one is set. hush-hush's /audit-log endpoint has no
// pagination parameters — QueryAuditLog always returns the full matching
// result set as a single slice, not a page.
type AuditLogFilter struct {
	ObjectID *string
	Caller   *string
	From     *time.Time
	To       *time.Time
}

// QueryAuditLog returns the audit log entries matching filter, oldest
// first, exactly as hush-hush returns them. No credential is required.
func (c *Client) QueryAuditLog(ctx context.Context, filter AuditLogFilter) ([]AuditLogEntry, error) {
	params := &genclient.QueryAuditLogParams{
		ObjectId: filter.ObjectID,
		Caller:   filter.Caller,
		From:     filter.From,
		To:       filter.To,
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
