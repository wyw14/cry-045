package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
)

type RevisionAnswer struct {
	RequestID string
	Revision  int
	IssueID   string
	Text      string
}

type RevisionAnswerBook struct {
	mu      sync.RWMutex
	answers map[string]RevisionAnswer
}

func NewRevisionAnswerBook() *RevisionAnswerBook {
	return &RevisionAnswerBook{answers: map[string]RevisionAnswer{}}
}

func revisionAnswerKey(requestID string, revision int, issueID string) string {
	return requestID + "\x00" + strconv.Itoa(revision) + "\x00" + issueID
}

func (b *RevisionAnswerBook) Apply(ctx context.Context, requestID string, revision int, answers []RevisionAnswer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	staged := make(map[string]RevisionAnswer, len(answers))
	for _, answer := range answers {
		if answer.RequestID != requestID || answer.Revision != revision {
			return errors.New("answer belongs to another revision")
		}
		if strings.TrimSpace(answer.Text) == "" {
			return errors.New("answer text is required")
		}
		staged[revisionAnswerKey(answer.RequestID, answer.Revision, answer.IssueID)] = answer
	}
	for key, answer := range staged {
		b.answers[key] = answer
	}
	return nil
}

func (b *RevisionAnswerBook) Get(requestID string, revision int, issueID string) (RevisionAnswer, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	answer, ok := b.answers[revisionAnswerKey(requestID, revision, issueID)]
	return answer, ok
}
