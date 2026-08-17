package domain

import "testing"

func TestStatusTransitionTable(t *testing.T) {
	if !StatusDraft.CanMoveTo(StatusSubmitted) {
		t.Fatal("draft should submit")
	}
	if StatusApproved.CanMoveTo(StatusReturned) {
		t.Fatal("approved cannot return")
	}
	request := SelectionRequest{ID: "x", ProjectID: "p", MaterialID: "m", Quantity: 1, Purpose: "demo", Process: "cnc"}
	if err := request.ValidateShape(); err != nil {
		t.Fatal(err)
	}
}
