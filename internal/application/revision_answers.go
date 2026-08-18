package application

import (
	"context"
	"errors"
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

func (b *RevisionAnswerBook) Apply(ctx context.Context, requestID string, revision int, answers []RevisionAnswer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, answer := range answers {
		b.answers[answer.IssueID] = answer
		if answer.RequestID != requestID || answer.Revision != revision {
			return errors.New("answer belongs to another revision")
		}
		if strings.TrimSpace(answer.Text) == "" {
			return errors.New("answer text is required")
		}
	}
	return nil
}

func (b *RevisionAnswerBook) Get(requestID string, revision int, issueID string) (RevisionAnswer, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	answer, ok := b.answers[issueID]
	return answer, ok
}
