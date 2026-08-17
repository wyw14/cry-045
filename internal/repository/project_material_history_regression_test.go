package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry045/internal/domain"
)

func TestApprovedReplacementPreservesOriginalReviewBasis(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	first := ProjectMaterialVersion{Item: domain.ProjectMaterial{ProjectID: "p1", RequestID: "r1", MaterialID: "m1", ApprovedRevision: 1, ImpactSummary: "initial"}, ApprovalBasisHash: "basis-one", RecordedAt: now}
	second := ProjectMaterialVersion{Item: domain.ProjectMaterial{ProjectID: "p1", RequestID: "r2", MaterialID: "m1", ApprovedRevision: 2, ImpactSummary: "certificate changed"}, ApprovalBasisHash: "basis-two", ReplacesRevision: 1, RecordedAt: now.Add(time.Hour)}
	if err := store.RecordProjectMaterialVersion(ctx, first, "approve-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordProjectMaterialVersion(ctx, second, "approve-2"); err != nil {
		t.Fatal(err)
	}
	history, err := store.ListProjectMaterialVersions(ctx, "p1", "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].ApprovalBasisHash != "basis-one" || history[1].ReplacesRevision != 1 {
		t.Fatalf("approved basis was overwritten: %#v", history)
	}
}
