package microscope

import (
	"testing"
	"time"
)

func TestGroupBatchByType(t *testing.T) {
	now := time.Now().UTC()
	batch := []Entry{
		{ID: "1", Type: TypeQuery, CreatedAt: now},
		{ID: "2", Type: TypeRequest, CreatedAt: now},
		{ID: "3", Type: TypeQuery, CreatedAt: now},
		{ID: "4", Type: TypeLog, CreatedAt: now},
	}

	groups := groupBatchByType(batch)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if groups[0].Type != TypeRequest || groups[0].Label != "Requests" || len(groups[0].Entries) != 1 {
		t.Fatalf("unexpected first group: %+v", groups[0])
	}
	if groups[1].Type != TypeQuery || len(groups[1].Entries) != 2 {
		t.Fatalf("unexpected query group: %+v", groups[1])
	}
	if groups[2].Type != TypeLog || len(groups[2].Entries) != 1 {
		t.Fatalf("unexpected log group: %+v", groups[2])
	}
}
