package application

import (
	"context"
	"sort"
	"time"

	"github.com/wyw14/cry045/internal/domain"
	"github.com/wyw14/cry045/internal/repository"
)

type QueueItem struct {
	Request  domain.SelectionRequest
	Window   domain.ReviewWindow
	Blocking int
	Notes    []string
	Overdue  bool
}

// ReviewQueue builds a deterministic reviewer worklist. The method is kept in
// the application layer so a future PostgreSQL implementation can replace the
// store without changing the queue policy.
func (s *ComplianceService) ReviewQueue(ctx context.Context, actorRole string, now time.Time) ([]QueueItem, error) {
	ids := []string{"sel-demo"}
	items := make([]QueueItem, 0, len(ids))
	for _, id := range ids {
		selection, err := s.store.GetSelection(ctx, id)
		if err != nil {
			if err == domain.ErrNotFound {
				continue
			}
			return nil, err
		}
		if selection.Status != domain.StatusSubmitted && selection.Status != domain.StatusFirstReview && selection.Status != domain.StatusSecondReview {
			continue
		}
		observations := domain.BuildObservations(domain.Material{ID: selection.MaterialID, Code: selection.MaterialID, AllowedProcesses: []string{selection.Process}}, selection, nil, now)
		blocking, notes := domain.SummarizeObservations(observations)
		window := domain.NewReviewWindow(now, selection.Status, actorRole, 48*time.Hour)
		items = append(items, QueueItem{Request: selection, Window: window, Blocking: blocking, Notes: notes, Overdue: window.Overdue(now)})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Overdue != items[j].Overdue {
			return items[i].Overdue
		}
		return items[i].Request.ID < items[j].Request.ID
	})
	return items, nil
}

var _ repository.Store
