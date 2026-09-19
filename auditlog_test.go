package hushhush_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go/v2"
)

func TestClient_QueryAuditLog_Filters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]hushhush.AuditLogEntry{
			{ObjectId: "obj-1", Action: hushhush.ActionRead, Timestamp: time.Unix(0, 0).UTC()},
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	objectID := "obj-1"
	entries, err := client.QueryAuditLog(context.Background(), hushhush.AuditLogFilter{ObjectID: &objectID})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 || entries[0].ObjectId != "obj-1" {
		t.Errorf("entries = %+v, want one entry for obj-1", entries)
	}
	if gotQuery != "object_id=obj-1" {
		t.Errorf("query = %q, want %q", gotQuery, "object_id=obj-1")
	}
}

func TestClient_QueryAuditLog_ActorAndPagingFilters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		actorType := hushhush.ActorTypeToken
		actorID := "tok-1"
		_ = json.NewEncoder(w).Encode([]hushhush.AuditLogEntry{
			{
				Id:        2,
				ObjectId:  "obj-1",
				Action:    hushhush.ActionRead,
				ActorType: &actorType,
				ActorId:   &actorID,
				Timestamp: time.Unix(0, 0).UTC(),
			},
		})
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	actor := "tok-1"
	after := int64(1)
	limit := int32(50)
	entries, err := client.QueryAuditLog(context.Background(), hushhush.AuditLogFilter{
		Actor: &actor,
		After: &after,
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 || entries[0].Id != 2 {
		t.Errorf("entries = %+v, want one entry with id 2", entries)
	}
	if want := "actor=tok-1&after=1&limit=50"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
}

func TestClient_QueryAuditLog_SinglePagePerCall(t *testing.T) {
	// QueryAuditLog returns whatever one page the server sends back — it
	// never loops on After itself. A caller wanting every entry pages
	// forward by hand using the previous page's last entry's Id.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		entries := make([]hushhush.AuditLogEntry, 250)
		for i := range entries {
			entries[i] = hushhush.AuditLogEntry{ObjectId: "obj", Action: hushhush.ActionRead, Timestamp: time.Unix(0, 0).UTC()}
		}
		_ = json.NewEncoder(w).Encode(entries)
	}))
	defer srv.Close()

	client, err := hushhush.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	entries, err := client.QueryAuditLog(context.Background(), hushhush.AuditLogFilter{})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 250 {
		t.Errorf("len(entries) = %d, want 250 returned in a single call", len(entries))
	}
}
