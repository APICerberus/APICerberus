package store

import (
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestAuditRepoBatchInsertAndFindByID(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}

	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer s.Close()

	repo := s.Audits()
	if repo == nil {
		t.Fatalf("expected audit repo")
	}

	entries := []AuditEntry{
		{
			ID:              "audit-1",
			RequestID:       "req-1",
			RouteID:         "route-1",
			Method:          "GET",
			Path:            "/api/users",
			StatusCode:      200,
			LatencyMS:       12,
			BytesIn:         10,
			BytesOut:        24,
			Blocked:         false,
			RequestHeaders:  map[string]any{"X-Req": "1"},
			ResponseHeaders: map[string]any{"X-Res": "ok"},
			CreatedAt:       time.Now().UTC(),
		},
		{
			ID:          "audit-2",
			RequestID:   "req-2",
			RouteID:     "route-2",
			Method:      "POST",
			Path:        "/api/orders",
			StatusCode:  403,
			Blocked:     true,
			BlockReason: "ip_blocked",
			RequestBody: `{"foo":"bar"}`,
			CreatedAt:   time.Now().UTC(),
		},
	}

	if err := repo.BatchInsert(entries); err != nil {
		t.Fatalf("BatchInsert error: %v", err)
	}

	found, err := repo.FindByID("audit-2")
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if found == nil {
		t.Fatalf("expected audit entry")
	}
	if !found.Blocked || found.BlockReason != "ip_blocked" {
		t.Fatalf("unexpected blocked fields: %+v", found)
	}
	if found.Method != "POST" {
		t.Fatalf("unexpected method: %s", found.Method)
	}
}

func TestAuditRepoListWithFilters(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}

	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer s.Close()

	repo := s.Audits()
	if err := repo.BatchInsert([]AuditEntry{
		{ID: "a1", UserID: "u1", RouteID: "r1", Method: "GET", StatusCode: 200, CreatedAt: time.Now().UTC()},
		{ID: "a2", UserID: "u1", RouteID: "r1", Method: "POST", StatusCode: 500, CreatedAt: time.Now().UTC().Add(time.Millisecond)},
		{ID: "a3", UserID: "u2", RouteID: "r2", Method: "GET", StatusCode: 404, CreatedAt: time.Now().UTC().Add(2 * time.Millisecond)},
	}); err != nil {
		t.Fatalf("BatchInsert error: %v", err)
	}

	result, err := repo.List(AuditListOptions{UserID: "u1", StatusMin: 400, Limit: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total=1 got %d", result.Total)
	}
	if len(result.Entries) != 1 || result.Entries[0].ID != "a2" {
		t.Fatalf("unexpected entries: %+v", result.Entries)
	}
}
