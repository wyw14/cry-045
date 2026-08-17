package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry045/internal/domain"
	"github.com/wyw14/cry045/internal/repository"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestDemoSelectionValidationAndApprovalFlow(t *testing.T) {
	now := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	store := repository.NewDemoStore(now)
	svc := NewComplianceService(store, fixedClock{t: now})
	sel, err := svc.ValidateSelection(context.Background(), "sel-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Findings) != 0 {
		t.Fatalf("demo material should be valid: %#v", sel.Findings)
	}
	if err := svc.Submit(context.Background(), "sel-demo", "designer-1"); err != nil {
		t.Fatal(err)
	}
	if err := svc.BeginReview(context.Background(), "sel-demo", "reviewer-1", false); err != nil {
		t.Fatal(err)
	}
	if err := svc.BeginReview(context.Background(), "sel-demo", "reviewer-2", true); err != nil {
		t.Fatal(err)
	}
	if err := svc.Approve(context.Background(), "sel-demo", "reviewer-2"); err != nil {
		t.Fatal(err)
	}
	project, err := store.GetProjectMaterial(context.Background(), "project-atlas", "mat-al")
	if err != nil || project.ApprovedRevision != 1 {
		t.Fatalf("approved project material missing: %#v %v", project, err)
	}
}

func TestReturnRequiresCurrentRevisionBeforeResubmit(t *testing.T) {
	now := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	store := repository.NewDemoStore(now)
	svc := NewComplianceService(store, fixedClock{t: now})
	if err := svc.Submit(context.Background(), "sel-demo", "d"); err != nil {
		t.Fatal(err)
	}
	if err := svc.BeginReview(context.Background(), "sel-demo", "r", false); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReturnForRevision(context.Background(), "sel-demo", "r", "补充 reach 证据"); err != nil {
		t.Fatal(err)
	}
	selection, err := svc.Revise(context.Background(), "sel-demo", "d", "机箱支架更新", "cnc", 130)
	if err != nil {
		t.Fatal(err)
	}
	if selection.Revision != 2 || selection.Status != domain.StatusDraft {
		t.Fatalf("unexpected revision: %#v", selection)
	}
}
