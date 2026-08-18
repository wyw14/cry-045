package application

import (
	"context"
	"testing"
)

func TestRevisionAnswersAreIsolatedAndCommittedAtomically(t *testing.T) {
	book := NewRevisionAnswerBook()
	ctx := context.Background()
	if err := book.Apply(ctx, "sel-demo", 1, []RevisionAnswer{{RequestID: "sel-demo", Revision: 1, IssueID: "issue-spec", Text: "first revision"}}); err != nil {
		t.Fatal(err)
	}
	err := book.Apply(ctx, "sel-demo", 2, []RevisionAnswer{
		{RequestID: "sel-demo", Revision: 2, IssueID: "issue-spec", Text: "second revision"},
		{RequestID: "sel-demo", Revision: 2, IssueID: "issue-proof", Text: ""},
	})
	if err == nil {
		t.Fatal("Apply succeeded with an incomplete issue answer")
	}
	old, ok := book.Get("sel-demo", 1, "issue-spec")
	if !ok || old.Text != "first revision" {
		t.Fatalf("revision 1 answer was overwritten: %#v, ok=%v", old, ok)
	}
	if _, ok := book.Get("sel-demo", 2, "issue-spec"); ok {
		t.Fatal("failed revision 2 batch was partially committed")
	}
}
