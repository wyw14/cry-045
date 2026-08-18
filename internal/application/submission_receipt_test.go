package application

import (
	"context"
	"errors"
	"testing"
)

func TestCancelledSubmissionLeavesNoPartialReceipts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	secondRan := false
	ledger := NewReceiptLedger(nil)
	service := &ComplianceService{}
	err := service.SubmitReceipts(ctx, ledger, "sel-demo", "designer-1", []ReceiptStep{
		{Name: "selection_saved", Run: func(context.Context) error { cancel(); return nil }},
		{Name: "submitter_notified", Run: func(context.Context) error { secondRan = true; return nil }},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit error = %v, want context.Canceled", err)
	}
	if secondRan {
		t.Fatal("step after cancellation was executed")
	}
	if got := ledger.Snapshot("sel-demo"); len(got) != 0 {
		t.Fatalf("partial receipts committed after cancellation: %#v", got)
	}
}
