package hushhush_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

// ListObjects and ListConsumerTokens fetch every row however many there are
// (alrayyes/hush-hush-go#228). The server pages with limit and offset and
// says the total in X-Total-Count; an older one ignores both and sends
// everything once, with no total.

// pagingServer serves rows[i] for i in [0, total) at path, paging like the
// real server when honour is true and ignoring limit and offset when not.
type pagingServer struct {
	*httptest.Server
	mu       sync.Mutex
	queries  []string
	failPage int // fail every request from the nth (1-based) on, 0 for never
}

func newPagingServer(t *testing.T, path string, total int, honour bool, row func(i int) any) *pagingServer {
	t.Helper()

	ps := &pagingServer{}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ps.mu.Lock()
		ps.queries = append(ps.queries, r.URL.RawQuery)
		n := len(ps.queries)
		ps.mu.Unlock()

		if r.URL.Path != path {
			http.NotFound(w, r)

			return
		}

		if ps.failPage != 0 && n >= ps.failPage { // the SDK retries a 5xx, so fail every request from here on
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		limit, offset := total, 0
		if honour {
			if v := r.URL.Query().Get("limit"); v != "" {
				limit, _ = strconv.Atoi(v)
			}

			if v := r.URL.Query().Get("offset"); v != "" {
				offset, _ = strconv.Atoi(v)
			}

			w.Header().Set("X-Total-Count", strconv.Itoa(total))
		}

		rows := []any{}
		for i := offset; i < min(offset+limit, total); i++ {
			rows = append(rows, row(i))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rows)
	}))
	t.Cleanup(ps.Close)

	return ps
}

func (ps *pagingServer) requests() []string {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return append([]string(nil), ps.queries...)
}

func objectRow(i int) any {
	return map[string]any{"id": fmt.Sprintf("00000000-0000-4000-8000-%012d", i), "slug": fmt.Sprintf("obj-%04d", i), "tags": []string{}}
}

func tokenRow(i int) any {
	return map[string]any{"id": fmt.Sprintf("tok-%04d", i), "consumer": "c", "description": "d", "created_at": "2026-01-01T00:00:00Z", "expires_at": "2027-01-01T00:00:00Z", "revoked": false}
}

func newListClient(t *testing.T, url string) *hushhush.Client {
	t.Helper()

	client, err := hushhush.NewClient(url, hushhush.WithAPIKey("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func TestClient_ListObjects_ReadsEveryPage(t *testing.T) {
	ps := newPagingServer(t, "/objects", 1200, true, objectRow)

	got, err := newListClient(t, ps.URL).ListObjects(context.Background(), "")
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}

	if len(got) != 1200 {
		t.Fatalf("got %d objects, want 1200", len(got))
	}

	for i, o := range got {
		if want := fmt.Sprintf("obj-%04d", i); o.Slug != want {
			t.Fatalf("object %d = %q, want %q (the server's order)", i, o.Slug, want)
		}
	}

	want := []string{"limit=500&offset=0", "limit=500&offset=500", "limit=500&offset=1000"}
	if got := ps.requests(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
}

func TestClient_ListObjects_AnOlderServerIsReadOnce(t *testing.T) {
	ps := newPagingServer(t, "/objects", 30, false, objectRow)

	got, err := newListClient(t, ps.URL).ListObjects(context.Background(), "")
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}

	if len(got) != 30 {
		t.Errorf("got %d objects, want 30", len(got))
	}

	if n := len(ps.requests()); n != 1 {
		t.Errorf("%d requests, want 1: a server that sends no total has nothing more to page", n)
	}
}

func TestClient_ListObjects_AFailedPageReturnsTheErrorAndNoRows(t *testing.T) {
	ps := newPagingServer(t, "/objects", 1200, true, objectRow)
	ps.failPage = 2

	got, err := newListClient(t, ps.URL).ListObjects(context.Background(), "")
	if err == nil {
		t.Fatal("want an error from the failed second page, got nil")
	}

	if got != nil {
		t.Errorf("got %d rows with the error, want none", len(got))
	}
}

func TestClient_ListObjects_StopsWhenTheContextIsCancelled(t *testing.T) {
	ps := newPagingServer(t, "/objects", 1200, true, objectRow)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newListClient(t, ps.URL).ListObjects(ctx, "")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestClient_ListObjectsFiltered_SendsTheFiltersOnEveryPage(t *testing.T) {
	ps := newPagingServer(t, "/objects", 700, true, objectRow)

	_, err := newListClient(t, ps.URL).ListObjectsFiltered(context.Background(), hushhush.ListObjectsFilter{UsedBy: "homelab", Tags: []string{"prod"}})
	if err != nil {
		t.Fatalf("ListObjectsFiltered: %v", err)
	}

	reqs := ps.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2", len(reqs))
	}

	for _, q := range reqs {
		if !contains(q, "used_by=homelab") || !contains(q, "tag=prod") {
			t.Errorf("query %q lost a filter", q)
		}
	}
}

func TestClient_ListConsumerTokens_ReadsEveryPage(t *testing.T) {
	ps := newPagingServer(t, "/consumer-tokens", 1200, true, tokenRow)

	got, err := newListClient(t, ps.URL).ListConsumerTokens(context.Background())
	if err != nil {
		t.Fatalf("ListConsumerTokens: %v", err)
	}

	if len(got) != 1200 {
		t.Errorf("got %d tokens, want 1200", len(got))
	}

	if n := len(ps.requests()); n != 3 {
		t.Errorf("%d requests, want 3 pages of 500", n)
	}
}

func TestClient_ListConsumerTokens_AnOlderServerIsReadOnce(t *testing.T) {
	ps := newPagingServer(t, "/consumer-tokens", 12, false, tokenRow)

	got, err := newListClient(t, ps.URL).ListConsumerTokens(context.Background())
	if err != nil {
		t.Fatalf("ListConsumerTokens: %v", err)
	}

	if len(got) != 12 || len(ps.requests()) != 1 {
		t.Errorf("got %d tokens in %d requests, want 12 in 1", len(got), len(ps.requests()))
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}

	return false
}
