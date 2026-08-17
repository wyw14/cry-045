package httptransport

import (
	"testing"
	"time"

	"github.com/wyw14/cry045/internal/domain"
)

func TestTimelineViewFiltersWindowAndRedactsReviewerDetails(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	events := []domain.AuditEvent{
		{ID: "old", Action: "draft", ActorID: "designer-secret", Before: "old-before", After: "old-after", Hash: "1111111111111111", CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "inside", Action: "return", ActorID: "reviewer-secret", Before: "private-before", After: "private-after", Hash: "2222222222222222", CreatedAt: now},
	}
	view := prepareTimelineView(events, "designer", now.Add(-time.Hour), now.Add(time.Hour))
	if len(view) != 1 || view[0].ID != "inside" {
		t.Fatalf("timeline ignored requested window: %#v", view)
	}
	if view[0].Actor != "reviewer" || view[0].Before != "" || view[0].After != "" {
		t.Fatalf("reviewer details leaked to submitter: %#v", view[0])
	}
}
