package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry045/internal/domain"
)

func TestApplyRevisionRejectsStaleSecondWriter(t *testing.T) {
	store := NewDemoStore(time.Now())
	first, err := store.ApplyRevision(context.Background(), "sel-demo", 1, "edit-a", func(selection *domain.SelectionRequest) error { selection.Quantity = 200; return nil })
	if err != nil || first.Revision != 2 {
		t.Fatalf("first update failed: %#v %v", first, err)
	}
	_, err = store.ApplyRevision(context.Background(), "sel-demo", 1, "edit-b", func(selection *domain.SelectionRequest) error { selection.Quantity = 300; return nil })
	if !errors.Is(err, domain.ErrStaleRevision) {
		t.Fatalf("stale writer was accepted: %v", err)
	}
	stored, _ := store.GetSelection(context.Background(), "sel-demo")
	if stored.Quantity != 200 || stored.Revision != 2 {
		t.Fatalf("stale writer overwrote current revision: %#v", stored)
	}
}
