package microscope

import (
	"testing"
	"time"
)

func TestGroupBatchRelatedExcludesCurrent(t *testing.T) {
	now := time.Now().UTC()
	batch := []Entry{
		{ID: "e1", Type: TypeRequest, CreatedAt: now},
		{ID: "e2", Type: TypeQuery, CreatedAt: now},
		{ID: "e3", Type: TypeLog, CreatedAt: now},
	}

	groups := groupBatchRelated(batch, "e1")
	if len(groups) != 2 {
		t.Fatalf("expected 2 related groups, got %d", len(groups))
	}
	if groups[0].Type != TypeQuery {
		t.Fatalf("expected queries first, got %s", groups[0].Type)
	}
}

func TestEntryContentTabsRequest(t *testing.T) {
	tabs := entryContentTabs(Entry{
		Type: TypeRequest,
		Content: map[string]any{
			"headers":       map[string][]string{"Accept": {"application/json"}},
			"response_body": `{"status":"ready"}`,
		},
	})
	if len(tabs) != 2 {
		t.Fatalf("expected headers and response tabs, got %d", len(tabs))
	}
}

func TestBatchGroupSummaryQueries(t *testing.T) {
	now := time.Now().UTC()
	summary := batchGroupSummary(BatchTypeGroup{
		Type: TypeQuery,
		Entries: []Entry{
			{ID: "1", Content: map[string]any{"sql": "SELECT 1"}, CreatedAt: now},
			{ID: "2", Content: map[string]any{"sql": "SELECT 1"}, CreatedAt: now},
		},
	})
	if summary != "2 queries, 1 of which are duplicated." {
		t.Fatalf("unexpected summary: %s", summary)
	}
}
