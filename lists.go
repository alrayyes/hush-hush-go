package hushhush

import (
	"context"
	"net/http"
	"strconv"
)

// listPageSize is how many rows each request for a list asks for: the most
// the server allows, so a long list takes as few requests as it can.
const listPageSize = 500

// page is one response of a paged list. total is what the server says there
// are in all (its X-Total-Count), and hasTotal is false for a server that
// sent none.
type page[T any] struct {
	rows     []T
	total    int
	hasTotal bool
}

// totalCount reads X-Total-Count. A server from before paging sends none.
func totalCount(h http.Header) (int, bool) {
	n, err := strconv.Atoi(h.Get("X-Total-Count"))

	return n, err == nil
}

// readAllPages calls fetch for one page at a time, from offset 0, until it
// has read every row the server says there are, and returns them in the
// server's order. A server that sends no total ignores limit and offset and
// has already sent everything, so it's read once and not asked again. A
// failed page returns its error and no rows, so a caller never sees a list
// that quietly stops short.
func readAllPages[T any](ctx context.Context, fetch func(ctx context.Context, limit, offset int32) (page[T], error)) ([]T, error) {
	var all []T

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		p, err := fetch(ctx, listPageSize, int32(len(all))) //nolint:gosec // len(all) stays far below MaxInt32: the server caps a total at it
		if err != nil {
			return nil, err
		}

		all = append(all, p.rows...)

		if !p.hasTotal || len(p.rows) == 0 || len(all) >= p.total {
			if all == nil {
				all = []T{}
			}

			return all, nil
		}
	}
}
